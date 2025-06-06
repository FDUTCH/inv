package inv

import (
	"github.com/bedrock-gophers/intercept/intercept"
	"github.com/bedrock-gophers/unsafe/unsafe"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"sync/atomic"
)

func init() {
	intercept.Hook(packetHandler{})
}

type packetHandler struct{}

func (h packetHandler) HandleClientPacket(ctx *intercept.Context, pk packet.Packet) {
	switch pk.(type) {
	case *packet.ItemStackRequest, *packet.ContainerClose:
	default:
		return
	}

	ha, _ := ctx.Val().Handle()
	ha.ExecWorld(func(tx *world.Tx, e world.Entity) {
		p := e.(*player.Player)
		s := unsafe.Session(p)
		switch pkt := pk.(type) {
		case *packet.ItemStackRequest:
			handleItemStackRequest(s, pkt.Requests)
		case *packet.ContainerClose:
			handleContainerClose(ctx, p, s, pkt.WindowID)
		}
	})
}

func (h packetHandler) HandleServerPacket(_ *intercept.Context, _ packet.Packet) {
	// Do nothing
}

func handleContainerClose(ctx *intercept.Context, p *player.Player, s *session.Session, windowID byte) {
	mn, ok := lastMenu(s)
	if !ok {
		return
	}
	currentID := fetchPrivateField[atomic.Uint32](s, "openedWindowID")
	if byte(currentID.Load()) == windowID && windowID == mn.windowID {
		closeLastMenu(p, mn)
		return
	}
	ctx.Cancel()
	p.OpenBlockContainer(mn.pos, p.Tx())
	closeLastMenu(p, mn)
}

func handleItemStackRequest(s *session.Session, req []protocol.ItemStackRequest) {
	for _, data := range req {
		for _, action := range data.Actions {
			updateActionContainerID(action, s)
		}
	}
}

// updateActionContainerID updates the container ID of the given action based on the current menu state.
// It is useful in case we use some unimplemented blocks such as hoppers.
func updateActionContainerID(action protocol.StackRequestAction, s *session.Session) {
	switch act := action.(type) {
	case *protocol.TakeStackRequestAction:
		if act.Source.Container.ContainerID != act.Destination.Container.ContainerID || act.Source.Container.ContainerID == protocol.ContainerCursor || act.Source.Container.ContainerID == protocol.ContainerHotBar {
			break
		}
		if _, ok := lastMenu(s); ok {
			act.Source.Container.ContainerID = protocol.ContainerLevelEntity
		}
	case *protocol.PlaceStackRequestAction:
		if act.Source.Container.ContainerID != act.Destination.Container.ContainerID || act.Source.Container.ContainerID == protocol.ContainerCursor || act.Source.Container.ContainerID == protocol.ContainerHotBar {
			break
		}
		if _, ok := lastMenu(s); ok {
			act.Source.Container.ContainerID = protocol.ContainerLevelEntity
		}
	case *protocol.DropStackRequestAction:
		if act.Source.Container.ContainerID == protocol.ContainerInventory || act.Source.Container.ContainerID == protocol.ContainerCursor || act.Source.Container.ContainerID == protocol.ContainerHotBar {
			break
		}
		if _, ok := lastMenu(s); ok {
			act.Source.Container.ContainerID = protocol.ContainerLevelEntity

		}
	case *protocol.SwapStackRequestAction:
		if act.Source.Container.ContainerID != act.Destination.Container.ContainerID || act.Source.Container.ContainerID == protocol.ContainerCursor || act.Source.Container.ContainerID == protocol.ContainerHotBar {
			break
		}
		if _, ok := lastMenu(s); ok {
			act.Source.Container.ContainerID = protocol.ContainerLevelEntity
		}
	}
}
