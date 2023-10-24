package provider

type Domain struct {
	Id        string
	Name      string
	Type      string
	Records   []string
	TTL       int
	ZoneId    string
	ZoneName  string
	FQDN      string
	Activated bool
}

type Zone struct {
	Id        string
	Name      string
	Activated bool
}
