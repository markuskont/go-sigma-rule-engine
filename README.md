# go-sigma-rule-engine

> Golang library that implements a sigma log rule parser and match engine.

[Sigma](https://github.com/Neo23x0/sigma) is a open and vendor-agnostic signature format for logs. Official sigma repository includes rule format definition, a public ruleset, and python tooling for converting rules into various SIEM alert formats. Essentially, it fills the same role in logging space as Suricata does in packet capture and YARA for file analysis. However, unlike those projects, the open Sigma project does not act as a match engine. Users are still expected to run a supported SIEM or log management solution, with necessary licencing to enable alerting features.

This project implements a rule parser and real-time match engine in Golang, to provide a lightweight alternative to those SIEM systems. Essentially, it's just a ~3000 line library that can be used by anyone to build their own IDS for logs. Initial version was experimental hack, cobbled together at the last minute with minimal testing, that was used by Crossed Swords 2020 exercise, organized by NATO CCDCOE. Yellow team log post-processor relied on the engine to detect Red team activities in gamenet targets, in real time. This code is archived into `pkg/sigma/v1`.

Since then, I rewrote the entire engine to provide a cleaner and more tested version as reference to anyone interested in building their own IDS for logs. This code can be found in `pkg/sigma/v2`. The project also includes a cli application in `cmd/` folder, written with [cobra](https://github.com/spf13/cobra). However, it is meant to be used as reference and testing, rather than a fully-fledged tool.

# Basic usage

Simply pull the code using `go get`.

```
go get -u github.com/markuskont/go-sigma-rule-engine/
```

Then import the library into your project. **PS** - this path was refactored to project root in `0.3`, as initial layout was not suitable to library project. Please update your imports accordingly when upgrading from `0.2`.

```go
import (
	"github.com/markuskont/go-sigma-rule-engine"
)
```

```go
ruleset, err := sigma.NewRuleset(sigma.Config{
  Directory: viper.GetStringSlice("rules.dir"),
})
if err != nil {
  return err
}
logrus.Debugf("Found %d files, %d ok, %d failed, %d unsupported",
  ruleset.Total, ruleset.Ok, ruleset.Failed, ruleset.Unsupported)
```

Events can then be evaluated against full ruleset.

```go
if result, match := ruleset.EvalAll(e); match {
  // handle match results here here
}
```

Individual rules could also be manually looped. For example, when early return is desired for avoiding full ruleset evaluation.

```go
for _, rule := range ruleset.Rules {
  if rule.Match(e) {
    // handle rule match here
  }
}
```

Note that variable `e` should implement `Event` interface.

## Matcher and Event

Our Sigma rule is built as a tree where each node must satisfy the `Matcher` interface that performs boolean evaluation for events.

```go
type Matcher interface {
	Match(Event) bool
}
```

There are simply too many possible event formats for our simple Sigma library to handle. Therefore, users are expected to implement `Event` interface for any object that will be matched against the ruleset. This `Event` interface embeds field access methods for two Sigma rule types - `keyword` and `selection`.

```go
// Keyworder implements keywords sigma rule type on arbitrary event
// Should return list of fields that are relevant for rule matching
type Keyworder interface {
	// Keywords implements Keyworder
	Keywords() ([]string, bool)
}

// Selector implements selection sigma rule type
type Selector interface {
	// Select implements Selector
	Select(string) (interface{}, bool)
}

// Event implements sigma rule types by embedding Keyworder and Selector
// Used by rules to extract relevant fields
type Event interface {
	Keyworder
	Selector
}
```

[Helper function source file](/pkg/sigma/v2/helpers.go) provides an example for handling dynamic hash maps.

### Keywords

`Keywords` rule type is simply a list of patters that must exist in core message. Only one pattern must match.

```yaml
  keywords:
    - 'wget * - http* | perl'
    - 'wget * - http* | sh'
    - 'wget * - http* | bash'
    - 'python -m SimpleHTTPServer'
```

Thus, the `Keyworder` interface simply returns a list of unstructured fields that could be considered core messages. It is built around `slice` because some event types, like Windows EventLog, could contain multiple fields that might contain this information. And returning a `nil` slice is cleaner than empty string when keyword rule type does not apply to event. However, in that case the second return value should always be `false` to ensure early return when rule does not apply to particular message.

```go
type Keyworder interface {
	Keywords() ([]string, bool)
}
```

Dynamic JSON objects can be implemented as stub because this rule type does not support key-value lookups.

```go
// Keywords implements Keyworder
func (s DynamicMap) Keywords() ([]string, bool) {
	return nil, false
}
```

Alternatively, structs for well-known and standardized messages, such as BSD syslog, might simply return the `Message` field.

```go
func (m Syslog) Keywords() ([]string, bool) {
	return m.Message.Keywords()
}
```

That message could be a unstructured string that also implements our `Event` interface.

```go
type Message string

func (m Message) Keywords() ([]string, bool) {
	return []string{string(m)}, true
}
```

Dynamic structured events, like Suricata EVE, could have well known fields that might qualify as message.

```go
func (s DynamicMap) Keywords() ([]string, bool) {
	if val, ok := s.Select("alert.signature"); ok {
		if str, ok := val.(string); ok {
			return []string{str}, true
		}
	}
	return nil, false
}
```

Multiple fields could be extracted and passed to the rule with this method. For example, `payload_printable`, `alert.category`, etc.

### Selection

This rule type is for key-value lookups.

```yaml
  selection:
    winlog.event_data.ScriptBlockText:
    - ' -FromBase64String'
```

Like with `keyword`, this rule type might simply may not apply to some events.

```go
func (s UnstructuredEvent) Select(key string) (interface{}, bool) {
	return nil, false
}
```

Otherwise, dynamic maps might simply implement it as wrapper for key-value lookup.

```go
func (s DynamicMap) Select(key string) (interface{}, bool) {
	if val, ok := d[key]; ok {
		return val, true
	}
	return nil, false
}
```

Static structs for well-standardized event formats may simply handle these lookups manually.

```go
type Syslog struct {
	Timestamp time.Time `json:"@timestamp"`
	Host      string    `json:"host"`
	Program   string    `json:"program"`
	Pid       int       `json:"pid"`
	Severity  int       `json:"severity"`
	Facility  int       `json:"facility"`
	Sender    net.IP    `json:"ip"`

	Message `json:"message"`
}

func (m Syslog) Select(key string) (interface{}, bool) {
	switch key {
	case "timestamp", "@timestamp":
		return m.Timestamp, true
	case "host":
		return m.Host, true
	case "program":
		return m.Program, true
	case "pid":
		return m.Pid, true
	case "severity":
		return m.Severity, true
	case "facility":
		return m.Facility, true
	case "sender":
		if m.Sender == nil {
			return nil, false
		}
		return m.Sender.String(), true
	case "message", "msg":
		return m.Keywords(), true
	default:
		return nil, false
	}
}
```

# Performance

```go
BenchmarkTreePositive0-12         867567              1363 ns/op
BenchmarkTreePositive1-12         862962              1494 ns/op
BenchmarkTreePositive2-12         795531              1380 ns/op
BenchmarkTreePositive3-12         854679              1393 ns/op
BenchmarkTreePositive4-12         884188              1364 ns/op
BenchmarkTreePositive5-12         809140              1390 ns/op
BenchmarkTreePositive6-12         773706              1410 ns/op
BenchmarkTreeNegative0-12         776173              1385 ns/op
BenchmarkTreeNegative1-12         812887              1481 ns/op
BenchmarkTreeNegative2-12         850477              1401 ns/op
BenchmarkTreeNegative3-12         840723              1390 ns/op
BenchmarkTreeNegative4-12         819126              1417 ns/op
BenchmarkTreeNegative5-12         748514              1416 ns/op
BenchmarkTreeNegative6-12         856683              1382 ns/op
```

# Limitations

**Ruleset is not thread safe**. Nor can it be easily deep-copied due to possible pointers behind interfaces and pattern containers. Each worker thread should instantiate independent ruleset. However, public sigma ruleset only produces about ~500 rules, so overhead is currently trivial.

**Library is built around distinct rules, rather than entire ruleset**. That means that each rule could run separate map lookups and no data is shared between them. While individual rules are quite efficient, even in current unoptimized form, passing each event thought entire ruleset means traversing hundreds of rules. Thus having significant performance overhead. For example, we measured that passing an ECS formatted Windows EventLog message through all Windows rules in public Sigma ruleset took 4.5 times the amount of time that was otherwise spent on simply decoding the message.

**Ruleset splitting and pre-filtering must be handled by the user.** Sigma has `logsource` field to indicate which events should be evaluated against a rule. We simply handled this externally, parsing rules into a map of smaller rulesets. So, we had separate rulesets for Syslog, Snoopy, Suricata and EventLog. Logsource field was used to determine which ruleset was executed for event.

**No support for aggregations or event correlation.** Relatively small amount of Sigma rules use aggregations with `count() > N` or `Near()` keywords. Implementing them in streaming scenario is quite complex, as they require sharing state between messages over sliding window. Thus requiring full event correlation to be implemented. However, this did not fit our present concurrency model where N workers load balance over common message channel and no information is shared between them. Future work.
