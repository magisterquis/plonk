Changelog
=========

v0.0.1-beta.2
-------------
- Changelog added
- Command-line flags changes (see `-h`'s output)
- Clearer documentation
- Default implant template now uses cURL's `-T.` instead of `--data-binary @-`
- Die with a warning if the user seems to have mixed up `-server` and client
  flags
- Default prompt now has a big `P`
- Don't allow IDless implant IDs
- Suggest `setcap` on Linux if listening fails
- Took a first whack at colorful output; set `PLONK_COLORIZE=true` in the Plonk
  client's environment to try it
