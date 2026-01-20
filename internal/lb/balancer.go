package lb

type Balancer interface {
	NextServer(cibles []string) []string 
}
