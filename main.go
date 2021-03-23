package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	//	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/kataras/tablewriter"
	"github.com/lensesio/tableprinter"
)

type Disk struct {
	Disk       string `header:"disk"`
	Md         string `header:"md"`
	Lvm        string `header:"lvm"`
	Zfs        string `header:"zfs"`
	Luks       string `header:"luks"`
	Size       string `header:"size"`
	Ssd        string `header:"ssd"`
	InUse      string `header:"in use"`
	LocalMount string `header:"localmount"`
	ID         string `header:"id"`
	Slot       string `header:"slot"`
}

// rows << ['Disk', 'md', 'lvm', 'zfs', 'luks', 'Size', 'SSD?', 'In Use', 'Local Mount?', 'ID', 'Slot']

type Blockdevice struct {
	Name       string `json:"name" header:"disk"`
	Fstype     string `json:"fstype"`
	Label      string `json:"label"`
	Uuid       string `json:"uuid"`
	Mountpoint string `json:"mountpoint" header:"mountpoint"`
	Md         string `header:"md"`
	Lvm        string `header:"lvm"`
	Zfs        string `header:"zfs"`
	Luks       string `header:"luks"`
	Size       string `header:"size"`
	Ssd        string `header:"ssd"`
	InUse      string `header:"in use"`
	ID         string `header:"id"`
	Slot       string `header:"slot"`
}

type Lsblk struct {
	Blockdevices []Blockdevice `json:"blockdevices"`
}

func main() {

	r, _ := regexp.Compile("^(sd[a-z]+|nvme[0-9]+n[0-9]+)$")

	blk := lsblk(r)

	for f := range blk.Blockdevices {
		blk.Blockdevices[f].Size, _ = getDiskSizeGB(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].Md := isMd(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].Lvm := isLvm(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].Zfs := isZfs(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].Luks := isLuks(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].Ssd := isSsd(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].inUse := inUse(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].id := id(blk.Blockdevices[f].Name)
		//blk.Blockdevices[f].slot := slot(blk.Blockdevices[f].Name)

	}

	printer := tableprinter.New(os.Stdout)
	sort.Slice(blk.Blockdevices, func(i, j int) bool {
		return blk.Blockdevices[j].Name > blk.Blockdevices[i].Name
	})
	printer.BorderTop, printer.BorderBottom, printer.BorderLeft, printer.BorderRight = true, true, true, true
	printer.CenterSeparator = "│"
	printer.ColumnSeparator = "│"
	printer.RowSeparator = "─"
	printer.HeaderBgColor = tablewriter.BgBlackColor
	printer.HeaderFgColor = tablewriter.FgGreenColor
	printer.DefaultAlignment = tableprinter.AlignRight // Set Alignment

	// Print the slice of structs as table, as shown above.
	printer.Print(blk.Blockdevices)

	//	fmt.Printf("\n\n\n")
	//	fmt.Printf("blk type: %T\n", blk)
	//	fmt.Printf("\n\n\n")
	//
	//	for bd := range blk.Blockdevices {
	//		fmt.Printf("%s\n", blk.Blockdevices[bd].Name)
	//	}
	//
}

func lsblk(r *regexp.Regexp) (out Lsblk) {

	lsblkOut, _ := exec.Command("lsblk", "--fs", "-J").Output()

	var tmpOut Lsblk

	json.Unmarshal([]byte(lsblkOut), &tmpOut)

	for d := range tmpOut.Blockdevices {
		if r.MatchString(tmpOut.Blockdevices[d].Name) {
			out.Blockdevices = append(out.Blockdevices, tmpOut.Blockdevices[d])
		}
	}

	fmt.Printf("out type: %T\n", tmpOut)
	fmt.Printf("length tmpout: %d\n", len(out.Blockdevices))
	return out
}

func isMd(device string) string {
	return "N"
}
func isLvm(device string) string {
	return "N"
}
func isZfs(device string) string {
	return "N"
}
func isLuks(device string) string {
	return "N"
}
func isSsd(device string) string {
	return "N"
}
func inUse(device string) string {
	return "N"
}
func localMount(device string) string {
	return "N"
}
func id(device string) string {
	return "N"
}
func slot(device string) string {
	return "N"
}

func getDevices(dir string, r *regexp.Regexp) (names []string, err error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return names, err
	}

	for _, file := range files {
		if r.MatchString(file.Name()) {
			names = append(names, file.Name())
		}
	}
	return names, err
}

func getDiskSizeGB(device string) (size string, err error) {
	f, _ := ioutil.ReadFile(fmt.Sprintf("/sys/block/%s/size", device))
	s := strings.Replace(string(f), "\n", "", -1)
	x, _ := strconv.Atoi(s)
	//fmt.Printf("Device size: %d\n", x)

	size = strconv.Itoa(x * 512 / 1000 / 1000 / 1000)
	//fmt.Printf("SIZE: %d\n", size)

	return size, err
}
