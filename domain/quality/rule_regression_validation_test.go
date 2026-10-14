package quality

import "testing"

func TestAuditDDDLimitValidation(t *testing.T) {
	cases := []struct {
		name  string
		field func(*Config) *Rule
	}{
		{"aggregate", func(c *Config) *Rule { return &c.Aggregate }},
		{"repository", func(c *Config) *Rule { return &c.Repository }},
		{"mutable_identity", func(c *Config) *Rule { return &c.MutableIdentity }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := DefaultConfig()
			tc.field(&c).Max = -1
			if err := c.Validate(); err == nil {
				t.Fatal("negative threshold accepted")
			}
		})
	}
}
