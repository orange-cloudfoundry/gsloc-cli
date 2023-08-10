package cli

type MemberMap struct {
	Ip       string `mapstructure:"ip"`
	Ratio    int    `mapstructure:"ratio"`
	DC       string `mapstructure:"dc"`
	Disabled bool   `mapstructure:"disabled"`
}
