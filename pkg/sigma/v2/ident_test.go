package sigma

var identCase1 = `
---
detection:
  condition: selection
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
`

var identCase2 = `
---
detection:
  condition: selection1 AND selection2
  selection1:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
  selection2:
    task: "Execute a Remote Command"
`

var identCase3 = `
---
detection:
  condition: selection1
  selection1:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
    task: "Execute a Remote Command"
`

var identCase4 = `
---
detection:
	condition: keywords
	keywords:
		- 'wget * - http* | perl'
		- 'wget * - http* | sh'
		- 'wget * - http* | bash'
		- 'python -m SimpleHTTPServer'
`
var identCase5 = `
---
detection:
	condition: selection
	selection:
		CommandLine|endswith: '.exe -S'
		ParentImage|endswith: '\services.exe'
`
