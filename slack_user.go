package main

const (
	Spring string = "XXXXXXXXX"
	Summer string = "XXXXXXXXX"
	Autumn string = "XXXXXXXXX"
	Winter string = "XXXXXXXXX"
)

func IsAdmin(id string) bool {
	admins := []string{
		Autumn,
	}
	return contains(admins, id)
}

func IsDeveploers(id string) bool {
	developers := []string{
		Spring,
		Summer,
		Autumn,
		Winter,
	}
	return contains(developers, id)
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}
