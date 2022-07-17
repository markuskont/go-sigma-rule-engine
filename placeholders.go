package sigma

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

type placeholder []string
type placeholderMap map[string]placeholder

type placeholderHandle struct {
	sync.RWMutex
	data placeholderMap
	path string
}

func (p *placeholderHandle) load() error {
	p.RWMutex.Lock()
	defer p.RWMutex.Unlock()
	f, err := os.Open(p.path)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewDecoder(f).Decode(&p.data)
}

func (p *placeholderHandle) runLoader(ctx context.Context, d time.Duration, errFn func(error)) error {
	if d == 0 {
		return errors.New("placeholder reloader requires a tick interval")
	}
	go func(ctx context.Context) {
		tick := time.NewTicker(d)
		defer tick.Stop()
	loop:
		for {
			select {
			case <-tick.C:
				if err := p.load(); err != nil && errFn != nil {
					errFn(err)
				}
			case <-ctx.Done():
				break loop
			}
		}
	}(ctx)
	return nil
}

func (p *placeholderHandle) matcher(key string) StringMatchers {
	p.RWMutex.RLock()
	defer p.RWMutex.RUnlock()
	m := make(StringMatchers, 0)
	if val, ok := p.data[key]; ok && val != nil {
		for _, pat := range val {
			// TODO - handle lowercase
			// whitespace squash should not be needed here, as it can lead to unexpected results for user
			m = append(m, ContentPattern{Token: pat})
		}
	}
	// missing key would lead to matcher with no values, which should always return false on match
	return m
}

func newPlaceholderHandle(confPath string) *placeholderHandle {
	return &placeholderHandle{
		RWMutex: sync.RWMutex{},
		data:    make(placeholderMap),
		path:    confPath,
	}
}

// updatePlaceholders walks the rule tree updates selection items that have placeholders defined
func updatePlaceholders(b Branch, ph *placeholderHandle) {
	switch t := b.(type) {
	case Selection:
		for _, item := range t.S {
			if item.Placeholder {
				item.update(ph.matcher(item.Key))
			}
		}
	case NodeNot:
		updatePlaceholders(t.B, ph)
	case NodeSimpleAnd:
		for _, b2 := range t {
			updatePlaceholders(b2, ph)
		}
	case NodeSimpleOr:
		for _, b2 := range t {
			updatePlaceholders(b2, ph)
		}
	case NodeAnd:
		updatePlaceholders(t.L, ph)
		updatePlaceholders(t.R, ph)
	case NodeOr:
		updatePlaceholders(t.L, ph)
		updatePlaceholders(t.R, ph)
	}
}

func updateRulesetPlaceholders(rs *Ruleset) {
	for _, rule := range rs.Rules {
		updatePlaceholders(rule.Root, rs.placeholders)
	}
}
