package jobtemplate

func GetConnectionOfJobAndJobTemplate(namespace, name string) string {
	return namespace + "." + name
}
