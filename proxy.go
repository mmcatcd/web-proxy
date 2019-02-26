package main

import "./proxy"

func main() {
	myProxy := proxy.New()
	myProxy.Start()
}
