package netdev

import (
	"fmt"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/yangjie500/cloud-ovs-agent/pkg/logger"
)

func sanitizeIfaceName(name string) string {
	name = strings.ReplaceAll(name, "_", "-")
	if len(name) > 15 {
		name = name[:15]
	}
	return name
}

func getLink(name string) (netlink.Link, bool, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		logger.Debugf("[netdev] getlink(%s) -> not found", name)
		return nil, false, nil
	}
	logger.Debugf("[netdev] getLink(%s) -> found (type=%T)", name, link)
	return link, true, nil
}

func ensureMTU(link netlink.Link, mtu int) error {
	if mtu <= 0 {
		return nil
	}

	current := link.Attrs().MTU
	if current == mtu {
		logger.Debugf("[netdev] MTU already %d for %s", mtu, link.Attrs().Name)
		return nil
	}

	logger.Debugf("[netdev] setting MTU %d for %s (old=%d)", mtu, link.Attrs().Name, current)
	if err := netlink.LinkSetMTU(link, mtu); err != nil {
		logger.Errorf("[netdev] failed to set MTU on %s: %v", link.Attrs().Name, err)
		return fmt.Errorf("set MTU on %s: %w", link.Attrs().Name, err)
	}
	return nil
}

func CreateTap(baseName string, mtu int, withVnetHdr bool) (string, error) {
	ifName := sanitizeIfaceName(baseName)
	logger.Debugf("[netdev] creating TAP %s (mtu=%d, vnetHdr=%t)", ifName, mtu, withVnetHdr)

	if link, exists, err := getLink(ifName); err != nil {
		logger.Errorf("[netdev] failed to check existing link %s: %v", ifName, err)
		return ifName, err
	} else if exists {
		if _, ok := link.(*netlink.Tuntap); !ok {
			logger.Errorf("[netdev] %s exists but is not a TAP (type=%T)", ifName, link)
			return ifName, fmt.Errorf("link %q exists but is not a TAP", ifName)
		}
		if err := ensureMTU(link, mtu); err != nil {
			return ifName, err
		}
		logger.Infof("[netdev] TAP %s already exists", ifName)
		return ifName, nil
	}

	flags := netlink.TUNTAP_NO_PI
	if withVnetHdr {
		flags |= netlink.TUNTAP_VNET_HDR
	}

	tap := &netlink.Tuntap{
		LinkAttrs: netlink.LinkAttrs{
			Name: ifName,
			MTU:  mtu,
		},
		Mode:  netlink.TUNTAP_MODE_TAP,
		Flags: flags,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		logger.Errorf("[netdev] failed to add TAP %s: %v", ifName, err)
		return ifName, fmt.Errorf("add TAP %s: %w", ifName, err)
	}

	logger.Infof("[netdev] created TAP %s", ifName)
	return ifName, nil
}

func DeleteLink(baseName string) error {
	ifName := sanitizeIfaceName(baseName)
	logger.Infof("[netdev] deleting link %s", ifName)

	link, exists, err := getLink(ifName)
	if err != nil {
		logger.Errorf("[netdev] failed to check link %s: %v", ifName, err)
		return err
	}

	if !exists {
		logger.Warnf("[netdev] link %s not found, nothing to delete", ifName)
		return nil
	}

	if err := netlink.LinkDel(link); err != nil {
		logger.Errorf("[netdev] failed to delete link %s: %v", ifName, err)
		return fmt.Errorf("delete link %s: %w", ifName, err)
	}

	logger.Infof("[netdev] deleted link %s", ifName)
	return nil

}

func SetLinkUp(baseName string) (string, error) {
	ifName := sanitizeIfaceName(baseName)
	logger.Infof("[netdev] setting link %s UP", ifName)

	link, exists, err := getLink(ifName)
	if err != nil {
		return ifName, err
	}

	if !exists {
		logger.Errorf("[netdev] link %s not found, nothing to set UP", ifName)
		return ifName, fmt.Errorf("link %s does not exists, try maybe creating it first", ifName)
	} else {
		if _, ok := link.(*netlink.Tuntap); !ok {
			logger.Errorf("[netdev] %s exists but is not a TAP", ifName)
			return ifName, fmt.Errorf("link %q exists but is not a TAP", ifName)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		logger.Errorf("[netdev] failed to bring %s UP: %v", ifName, err)
		return ifName, fmt.Errorf("link up %s: %w", ifName, err)
	}

	logger.Infof("[netdev] link %s is UP", ifName)
	return ifName, nil
}

func SetLinkDown(baseName string) error {
	ifName := sanitizeIfaceName(baseName)
	logger.Infof("[netdev] setting link %s DOWN", ifName)

	link, exists, err := getLink(ifName)
	if err != nil {
		return err
	}
	if !exists {
		logger.Errorf("[netdev] link %s not found, nothing to set down", ifName)
		return fmt.Errorf("link %s does not exists, try maybe creating it first", ifName)
	}
	if err := netlink.LinkSetDown(link); err != nil {
		logger.Errorf("[netdev] failed to bring %s DOWN: %v", ifName, err)
		return fmt.Errorf("link down %s: %w", ifName, err)
	}

	logger.Infof("[netdev] link %s is DOWN", ifName)
	return nil
}
