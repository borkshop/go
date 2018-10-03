package main

import (
	"fmt"

	"børk.com/borkgen"
)

func main() {
	for hi := 0; hi < 7; hi++ {
		desc := borkgen.DescribeRoom(borkgen.Hilbert.Decode(hi))
		fmt.Printf("AT %s\n", desc.At)
		fmt.Printf("HI %d\n", desc.Hilbert)
		fmt.Printf("NX %s\n", desc.Next)
		fmt.Printf("PR %s\n", desc.Prev)
		fmt.Printf("SZ %s\n", desc.Size)
		fmt.Printf("           %2d\n", desc.NorthMargin)
		fmt.Printf("MARGINS %2d  X %2d\n", desc.WestMargin, desc.EastMargin)
		fmt.Printf("           %2d\n", desc.SouthMargin)
		fmt.Printf("\n")
	}
}
