package command

import (
	"flag"
	"fmt"
	"github.com/mitchellh/cli"
	"net"
	"regexp"
	"strings"
)

// MembersCommand is a Command implementation that queries a running
// Serf agent what members are part of the cluster currently.
type MembersCommand struct {
	Ui cli.Ui
}

func (c *MembersCommand) Help() string {
	helpText := `
Usage: serf members [options]

  Outputs the members of a running Serf agent.

Options:

  -detailed                 Additional information such as protocol verions
                            will be shown.

  -role=<regexp>            If provided, output is filtered to only nodes matching
                            the regular expression for role

  -rpc-addr=127.0.0.1:7373  RPC address of the Serf agent.

  -status=<regexp>			If provided, output is filtered to only nodes matching
                            the regular expression for status
`
	return strings.TrimSpace(helpText)
}

func (c *MembersCommand) Run(args []string) int {
	var detailed bool
	var roleFilter, statusFilter string
	cmdFlags := flag.NewFlagSet("members", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	cmdFlags.BoolVar(&detailed, "detailed", false, "detailed output")
	cmdFlags.StringVar(&roleFilter, "role", ".*", "role filter")
	cmdFlags.StringVar(&statusFilter, "status", ".*", "status filter")
	rpcAddr := RPCAddrFlag(cmdFlags)
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Compile the regexp
	roleRe, err := regexp.Compile(roleFilter)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to compile role regexp: %v", err))
		return 1
	}
	statusRe, err := regexp.Compile(statusFilter)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to compile status regexp: %v", err))
		return 1
	}

	client, err := RPCClient(*rpcAddr)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error connecting to Serf agent: %s", err))
		return 1
	}
	defer client.Close()

	members, err := client.Members()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error retrieving members: %s", err))
		return 1
	}

	for _, member := range members {
		// Skip the non-matching members
		if !roleRe.MatchString(member.Role) || !statusRe.MatchString(member.Status) {
			continue
		}

		addr := net.TCPAddr{IP: member.Addr, Port: int(member.Port)}
		c.Ui.Output(fmt.Sprintf("%s    %s    %s    %s",
			member.Name, addr.String(), member.Status, member.Role))

		if detailed {
			c.Ui.Output(fmt.Sprintf("    Protocol Version: %d",
				member.DelegateCur))
			c.Ui.Output(fmt.Sprintf("    Available Protocol Range: [%d, %d]",
				member.DelegateMin, member.DelegateMax))
		}
	}

	return 0
}

func (c *MembersCommand) Synopsis() string {
	return "Lists the members of a Serf cluster"
}
