package lb

type Balancer interface {
	Next() string
}
