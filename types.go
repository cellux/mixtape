package main

import (
	"image"
	"log"
)

type Point = image.Point
type Size = image.Point
type Rect = image.Rectangle

type Smp = float64

type SampleIterator func() Smp

type SampleIteratorProvider interface {
	GetSampleIterator() SampleIterator
}

func GetSampleIterator(val Val) SampleIterator {
	if provider, ok := val.(SampleIteratorProvider); ok {
		return provider.GetSampleIterator()
	}
	log.Fatalf("value of type %T does not implement SampleIteratorProvider", val)
	return nil
}
