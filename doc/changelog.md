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
- Added `make install`
- Output is now displayed before the request to `/o` finishes, if it's taking
  long enough, for use with single output streams; watch out for `ulimit -n`
- Append a newline to tasking; save humanity from more awkward loops
- Removed `,name` command, as it was more confusing than helpful
- Removed `,task` command, as it was more confusing than `,name`
- Unborked `,help topics`, which all now appear and are even sorted sensibly
- Simplified commands altogether
- Added ,i -next|-last for easier implant selection
- Added `,h` topics for other commands
