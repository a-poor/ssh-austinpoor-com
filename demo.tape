# Record the deployed site. Verify its SSH host key once before running this tape.
# Run from the repository root: vhs demo.tape
Output assets/demo.gif
Require ssh

Set Shell "bash"
Set FontSize 20
Set Width 1200
Set Height 780
Set Padding 24
Set Theme "Catppuccin Mocha"
Set Framerate 15
Set CursorBlink false
Set TypingSpeed 70ms
# Start on Home so the README preview shows the app rather than an empty prompt.
Set LoopOffset 12%

# Keep the shell prompt consistent and make unexpected SSH prompts fail.
Hide
Type@1ms "unset PROMPT_COMMAND; export PS1='$ '"
Enter
Type@1ms 'ssh() { command ssh -o BatchMode=yes -o StrictHostKeyChecking=yes "$@"; }'
Enter
Type@1ms "clear"
Enter
Sleep 300ms
Show

Type "ssh ssh.austinpoor.com"
Sleep 500ms
Enter
Hide
Wait+Screen@60s /Hey! I'm Austin\./
Show
Sleep 3s

# About, then browse the blog.
Tab
Wait+Screen /About me/
Sleep 2s
Tab
Wait+Screen /[0-9]+ posts/
Sleep 1500ms
Down@250ms 2
Sleep 1s

# Search by title/tag, apply the filter, and open the matching article.
Type "/"
Wait+Screen /Filter:/
Type@180ms "deno"
Enter
Wait+Screen /1 post/
Sleep 1500ms
Enter
Wait+Screen /Read on the website/
Sleep 3s
PageDown
Sleep 3s

# Return to the results, then visit the project page.
Escape
Wait+Screen /1 post/
Sleep 750ms
Type "4"
Wait+Screen /Wish/
Sleep 3s
Type "1"
Wait+Screen /Hey! I'm Austin\./
Sleep 1500ms
Type "q"
Wait+Screen /Connection to ssh\.austinpoor\.com closed/
Sleep 750ms
