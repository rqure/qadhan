package main

type NameProvider struct{}

func (n *NameProvider) Get() string {
	return "prayer"
}
