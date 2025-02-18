package main

import "fmt"

func filter(arr []string) []string {
	slice := []string{}
	seen := make(map[string]string)

	for _, s := range arr {
		curr, exists := seen[s]
		if exists {
			continue
		}
		seen[s] = curr
		slice = append(slice, s)

	}
	return slice
}

func logSlice(slice []string) {
	for _, s := range slice {
		fmt.Println(s)
	}
}
