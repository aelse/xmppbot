package main

type Command func(string) string

func ping(string) string {
	return "pong"
}
