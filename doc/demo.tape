# Demo for GitHub Banner
Output doc/demo.gif

Require echo

Set Shell "fish"
Set FontSize 24
Set Width 1280
Set Height 640

Type "echo Get Actions Usage for Current Repo | lolcat" Enter 2
Sleep 500ms
Type "gh actions-usage" Sleep 500ms Enter
Sleep 5s
Type "clear" Enter

Type "echo 'Get Actions Usage for Specified Repo(s)' | lolcat" Enter 2
Sleep 500ms
Type "gh actions-usage codiform/gh-actions-usage" Sleep 500ms Enter
Sleep 4s
Type "gh actions-usage geoffreywiseman/gh-actuse codiform/gh-actions-usage" Sleep 500ms Enter
Sleep 8s
Type "clear" Enter

Type "echo 'Get Actions Usage for Specified User / Organization' | lolcat" Enter 2
Sleep 500ms
Type "gh actions-usage kkruszewska" Sleep 500ms Enter
Sleep 4s
Type "gh actions-usage codiform" Sleep 500ms Enter
Sleep 8s
Type "clear" Enter

Type "echo 'Mix it up! Repos, Users, Orgs, Oh My!' | lolcat" Enter 2
Sleep 500ms
Type "gh actions-usage --skip geoffreywiseman codiform charmbracelet/vhs" Sleep 500ms Enter
Sleep 8s

# End
Sleep 10s