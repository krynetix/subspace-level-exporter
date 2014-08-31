package main

import (
	"fmt"
	"os"
	"errors"
	"encoding/binary"
	"bytes"
	"strings"
	"encoding/json"
	"strconv"
)

var directory string

var mapData = map[string]map[string]uint32{}

func exportTileset(fh *os.File) (uint32, error) {
	headerBuf := make([]byte, 2)
	sizeBuf := make([]byte, 4)

	n, err := fh.ReadAt(headerBuf, 0)
	if err != nil {
		fmt.Printf("Error trying to read header: %s\n", err)
		return 0, nil
	}

	if n != 2 || headerBuf[0] != 'B' || headerBuf[1] != 'M' {
		fmt.Println("No tileset found")
		return 0, nil
	}

	n, err = fh.ReadAt(sizeBuf, 2)
	if err != nil {
		fmt.Printf("Error trying to read tileset size: %s\n", err)
		return 0, nil
	}

	if n != 4 {
		return 0, errors.New("corrupt level format: no tileset size found")
	}


	var size uint32
	err = binary.Read(bytes.NewBuffer(sizeBuf), binary.LittleEndian, &size)
	if err != nil {
		return 0, errors.New("corrupt level format: tileset size corrupt")
	}

	bmpBuffer := make([]byte, size)
	n, err = fh.ReadAt(bmpBuffer, 0)
	if err != nil {
		return 0, fmt.Errorf("corrupt level format: could not read bitmap data, %s\n", err)
	}

	if uint32(n) != size {
		return 0, fmt.Errorf("corrupt level format: could not read bitmap data, expected %d bytes, got %d\n", size, n)
	}

	outFh, err := os.Create(directory + "/tiles.bmp")
	if err != nil {
		return 0, fmt.Errorf("could not create tiles.bmp: %s\n", err)
	}
	defer outFh.Close()
	
	n, err = outFh.Write(bmpBuffer)
	if err != nil {
		return 0, fmt.Errorf("could not write to tiles.bmp: %s\n", err)
	}

	if uint32(n) != size {
		return 0, fmt.Errorf("error writing to tiles.bmp, expected to write %d bytes, did %d\n", size, n)
	}

	

	return size, nil
}

func addTile(tileId, x, y uint32) {
	if _, ok := mapData[strconv.Itoa(int(x))]; !ok {
		mapData[strconv.Itoa(int(x))] = map[string]uint32{}
	}

	mapData[strconv.Itoa(int(x))][strconv.Itoa(int(y))] = tileId
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Please specify a filename")
		return
	}

	fileIn := os.Args[1]

	fh, err := os.Open(fileIn)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fh.Close()

	directory = strings.TrimSuffix(fileIn, ".lvl")
	if directory == fileIn {
		directory = directory + "-exported"
	}

	if _, err = os.Stat(directory); err != nil {

		err = os.Mkdir(directory, 0755)
		if err != nil {
			fmt.Printf("could not create output directory: %s\n", err)
			return
		}

	}

	start, err := exportTileset(fh)
	if err != nil {
		fmt.Println(err)
		return
	}

	newPos, err := fh.Seek(int64(start), 0)
	if err != nil {
		fmt.Println("could not seek to map data")
		return
	}

	if uint32(newPos) != start {
		fmt.Printf("could not seek to map data, should start at %d, we are at %d\n", start, newPos)
		return
	}
	
	buf := make([]byte, 4)
	n, err := fh.Read(buf)
	for n != 0 {
		if err != nil {
			fmt.Printf("error reading map data %s\n", err)
			return
		}

		if n != 4 {
			fmt.Printf("error reading map expected to read 4 bytes, but got %d\n", n)
			return
		}

		var tileData = binary.LittleEndian.Uint32(buf)
		
		var tile = tileData >> 24
		var x = tileData << 20 >> 20
		var y = tileData << 8 >> 20

		addTile(tile - 1, x, y)

		n, err = fh.Read(buf)
	}

	json, err := json.Marshal(mapData)
	if err != nil {
		fmt.Printf("error generating json: %s\n", err)
		return
	}

	jsonFile, err := os.Create(directory + "/map.json")
	if err != nil {
		return
	}
	defer jsonFile.Close()
	
	n, err = jsonFile.Write(json)
	if err != nil {
		fmt.Printf("could not write to map.json: %s\n", err)
		return
	}
}
