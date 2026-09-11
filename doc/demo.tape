# Demo for GitHub Banner
Output doc/demo.gif

Require echo
Require lolcat

Set Shell "fish"
Set FontSize 24
Set Width 1280
Set Height 640

# VHS captures frames by screenshotting the terminal; at higher framerates the capture
# falls behind and the recording plays faster than real time. 5 fps keeps up.
Set Framerate 5

# Each query is typed on camera, then recording pauses until the prompt returns,
# so the GIF skips the seconds spent fetching check runs from the API. VHS runs
# fish with its default "> " prompt; the pattern matches that prompt sitting idle.
Set WaitTimeout 120s
Set WaitPattern /^>\s*$/

Type "echo Get Actions Usage for Current Repo | lolcat" Enter 2
Sleep 1500ms
Type "gh actions-usage" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 7s
Type "clear" Enter

Type "echo 'Get Actions Usage for Specified Repo(s)' | lolcat" Enter 2
Sleep 1500ms
Type "gh actions-usage codiform/gh-actions-usage" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 6s
Type "gh actions-usage geoffreywiseman/gh-actuse codiform/gh-actions-usage" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 8s
Type "clear" Enter

Type "echo 'Get Actions Usage for Specified User / Organization' | lolcat" Enter 2
Sleep 1500ms
Type "gh actions-usage kkruszewska" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 6s
Type "gh actions-usage codiform" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 8s
Type "clear" Enter

Type "echo 'Mix it up! Repos, Users, Orgs, Oh My!' | lolcat" Enter 2
Sleep 1500ms
Type "gh actions-usage --skip geoffreywiseman codiform charmbracelet/vhs" Sleep 500ms Enter
Hide
Sleep 1s
Wait
Show
Sleep 10s

# End
Sleep 5s
