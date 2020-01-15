package sigma

/*
	This package will be the main entrypoint when imported from other projects.
*/

type Config struct {
}

func (c *Config) Validate() error {
	return nil
}

type Ruleset struct {
}

func NewRuleset(c *Config) (*Ruleset, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	r := &Ruleset{}
	return r, nil
}
