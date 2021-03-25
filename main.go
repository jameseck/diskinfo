package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
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

// rows << ['Disk', 'md', 'lvm', 'zfs', 'luks', 'Size', 'SSD?', 'In Use', 'Local Mount?', 'ID', 'Slot']

type Blockdevice struct {
	Name        string        `json:"name" header:"disk"`
	Fstype      string        `json:"fstype"`
	Label       string        `json:"label"`
	Uuid        string        `json:"uuid"`
	Mountpoint  string        `json:"mountpoint" header:"mountpoints"`
	Md          string        `header:"md"`
	Lvm         string        `header:"lvm"`
	Zfs         string        `header:"zfs"`
	Luks        string        `header:"luks"`
	Size        string        `header:"size,number"`
	Ssd         string        `header:"ssd/nvme"`
	InUse       string        `header:"in use"`
	VendorModel string        `header:"vendor/model"`
	Slot        string        `header:"slot"`
	Children    []Blockdevice `json:"children"`
}

type Lsblk struct {
	Blockdevices []Blockdevice `json:"blockdevices"`
}

func main() {

	r, _ := regexp.Compile("^(sd[a-z]+|nvme[0-9]+n[0-9]+)$")

	blk := lsblk(r)

	// Gather enclosure info
	enc, _ := filepath.Glob("/sys/class/enclosure/*/*/device/block/*")

	var enclosureDevices map[string]string
	enclosureDevices = make(map[string]string)

	for d := range enc {
		encS := strings.Split(enc[d], "/")
		encAddr := encS[4]
		deviceSlot := encS[5]
		deviceName := encS[8]
		enclosureDevices[deviceName] = fmt.Sprintf("%s/%s", encAddr, deviceSlot)
	}
	// TODO: Maybe extract enclosure model instead of PCI addr

	for f := range blk.Blockdevices {
		blk.Blockdevices[f].Size, _ = getDiskSizeGB(blk.Blockdevices[f].Name)
		if blk.Blockdevices[f].Fstype == "zfs_member" {
			blk.Blockdevices[f].Zfs = blk.Blockdevices[f].Label
		}

		if blk.Blockdevices[f].Fstype == "linux_raid_member" {
			blk.Blockdevices[f].Md = fmt.Sprintf("md%s", blk.Blockdevices[f].Label)
		}

		blk.Blockdevices[f].Lvm = GetLVM(&blk.Blockdevices[f])

		blk.Blockdevices[f].Mountpoint = GetMountpoints(&blk.Blockdevices[f])

		y, _ := ioutil.ReadFile(fmt.Sprintf("/sys/block/%s/queue/rotational", blk.Blockdevices[f].Name))
		s := strings.Replace(string(y), "\n", "", -1)
		if s == "1" {
			blk.Blockdevices[f].Ssd = ""
		} else {
			blk.Blockdevices[f].Ssd = "Y"
		}

		xv, _ := ioutil.ReadFile(fmt.Sprintf("/sys/block/%s/device/vendor", blk.Blockdevices[f].Name))
		xvs := strings.Replace(string(xv), "\n", "", -1)
		xm, _ := ioutil.ReadFile(fmt.Sprintf("/sys/block/%s/device/model", blk.Blockdevices[f].Name))
		xms := strings.Replace(string(xm), "\n", "", -1)
		blk.Blockdevices[f].VendorModel = fmt.Sprintf("%s %s", xvs, xms)

		for c := range blk.Blockdevices[f].Children {
			if blk.Blockdevices[f].Children[c].Fstype == "zfs_member" {
				blk.Blockdevices[f].Zfs = blk.Blockdevices[f].Children[c].Label
			}
		}

		blk.Blockdevices[f].Luks = GetLuks(&blk.Blockdevices[f])

		if _, ok := enclosureDevices[blk.Blockdevices[f].Name]; ok {
			blk.Blockdevices[f].Slot = enclosureDevices[blk.Blockdevices[f].Name]
		}

		blk.Blockdevices[f].InUse = InUse(&blk.Blockdevices[f])

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
	printer.DefaultAlignment = tableprinter.AlignLeft // Set Alignment

	// Print the slice of structs as table, as shown above.
	printer.Print(blk.Blockdevices)

}

func GetMountpoints(device *Blockdevice) string {
	var mounts []string
	if device.Mountpoint != "" {
		mounts = append(mounts, device.Mountpoint)
	}
	for c := range device.Children {
		if device.Children[c].Mountpoint != "" {
			mounts = append(mounts, device.Children[c].Mountpoint)
		}
	}
	return strings.Join(mounts, ",")
}

func GetLVM(device *Blockdevice) string {
	var vgs []string
	for c := range device.Children {
		if device.Children[c].Fstype == "LVM2_member" {
			pv, _ := exec.Command("pvs", "--separator", ",", "-o", "pv_name,vg_name", "--noheadings", fmt.Sprintf("/dev/%s", device.Children[c].Name)).Output()
			vgs = append(vgs, strings.Split(strings.Replace(string(pv), "\n", "", -1), ",")[1])
		}
	}
	return strings.Join(vgs, ",")
}

func GetLuks(device *Blockdevice) string {

	//	_, err := exec.Command("cryptsetup", "luksDump", fmt.Sprintf("/dev/%s", dev.Name)).Output()
	//	if err != nil {
	//		return ""
	//	} else {
	//		return "Y"
	//	}
	var lDevs []string
	if device.Fstype == "crypto_LUKS" {
		for c := range device.Children {
			lDevs = append(lDevs, device.Children[c].Name)
		}
	}
	return strings.Join(lDevs, ",")
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
	return out
}

func InUse(device *Blockdevice) string {
	// Check if luks, md, pv, zfs, etc etc
	if (device.Zfs != "") || (device.Label != "") || device.Fstype != "" || device.Mountpoint != "" || device.Md != "" || device.Lvm != "" || device.Luks != "" {
		return "Y"
	}
	return ""
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
	size = strconv.Itoa(x * 512 / 1000 / 1000 / 1000)
	return size, err
}
