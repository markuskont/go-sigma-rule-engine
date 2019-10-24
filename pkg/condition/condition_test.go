package condition

import (
	"fmt"
	"testing"
)

var data = []string{
	"aaa",
	"bbb1",
	"cccc*",
	"1 of fields*",
	"all of them",
	"aaa or bbb",
	"aaa and bbb",
	"aaa and (bbb or ccc)",
	"(1 of binary_*) and selection",
	`selection | count() by MachineName > 5`,
	`selection | count(dns_query) by parent_domain > 1000`,
}

func TestLex(t *testing.T) {
	for _, str := range data {
		l := lex(str)
		out := make([]Item, 0)
		for tok := range l.items {
			out = append(out, tok)
		}
		fmt.Printf("%+v\n", out)
	}
}
