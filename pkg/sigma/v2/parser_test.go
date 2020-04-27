package sigma

import "testing"

var detection1 = `
detection:
	condition: "selection1 and not selection3"
	selection1:
		Image:
			- *\schtasks.exe
			- *\nslookup.exe
			- *\certutil.exe
			- *\bitsadmin.exe
			- *\mshta.exe
		ParentImage:
			- *\mshta.exe
			- *\powershell.exe
			- *\cmd.exe
			- *\rundll32.exe
			- *\cscript.exe
			- *\wscript.exe
			- *\wmiprvse.exe
	selection3:
		CommandLine: "+R +H +S +A *.cui"
`

var detection1_positive = `
{
	"Image":       C:\test\bitsadmin.exe,
	"CommandLine": +R +H +A asd.cui,
	"ParentImage": C:\test\wmiprvse.exe,
	"Image":       C:\test\bitsadmin.exe,
	"CommandLine": aaa,
	"ParentImage": C:\test\wmiprvse.exe,
}
`

var detection1_negative = `
{
	"Image":       "C:\test\bitsadmin.exe",
	"CommandLine": "+R +H +S +A lll.cui",
	"ParentImage": "C:\test\mshta.exe"
}
`

type parseTestCase struct {
	Rule, Pos, Neg string
}

var parseTestCases = []parseTestCase{
	{Rule: detection1, Pos: detection1_positive, Neg: detection1_negative},
}

func TestTokenCollect(t *testing.T) {
	for _, c := range LexPosCases {
		p := &parser{
			lex: lex(c.Expr),
		}
		if err := p.collect(); err != nil {
			switch err.(type) {
			case ErrUnsupportedToken:
			default:
				t.Fatal(err)
			}
		}
	}
}
