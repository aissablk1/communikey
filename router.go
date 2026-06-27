package main

// router.go — couche 4 (routeur) : choisit le chemin de livraison.
//
// Principe directeur du bus : l'inbox coopératif est la voie PRÉFÉRÉE (fiable,
// durable, tout OS/provider) ; l'injection clavier live n'est qu'un REPLI pour
// les TUI Unix « muettes » ; offline → mise en file. Cf. la matrice §4 du design.

// Channel is the chosen delivery path.
type Channel int

const (
	ChannelInbox   Channel = iota // cooperative mailbox (preferred, universal)
	ChannelInject                 // live keystroke injection (Unix fallback)
	ChannelBridge                 // native bridge (e.g. Agent Teams mailbox)
	ChannelQueue                  // target offline → queue until reconnect
)

func (c Channel) String() string {
	switch c {
	case ChannelInbox:
		return "inbox"
	case ChannelInject:
		return "inject"
	case ChannelBridge:
		return "bridge"
	case ChannelQueue:
		return "queue"
	default:
		return "?"
	}
}

// TargetInfo is what the router needs to decide. It is deliberately transport- and
// provider-agnostic: discovery fills it in.
type TargetInfo struct {
	Cooperates bool // has a csend inbox/identity (can receive cooperatively)
	Bridge     bool // reachable only via a native bridge (Agent Teams in-process)
	Live       bool // a reachable TUI we can inject into right now (Unix multiplexer)
}

// Route applies the architecture's delivery priority:
//
//	cooperative inbox  >  native bridge  >  live injection  >  queue (offline)
//
// Inbox wins even when the target is also Live, because the mailbox is durable and
// race-free whereas injection is best-effort and terminal-specific.
func Route(t TargetInfo) Channel {
	switch {
	case t.Cooperates:
		return ChannelInbox
	case t.Bridge:
		return ChannelBridge
	case t.Live:
		return ChannelInject
	default:
		return ChannelQueue
	}
}
