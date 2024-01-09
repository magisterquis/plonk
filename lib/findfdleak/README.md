Find File Descriptor Leaks
==========================
Small program which wraps [kdump](https://man.openbsd.org/kdump) to work out
which files have not been closed.  Prints out a list of unclosed files.
