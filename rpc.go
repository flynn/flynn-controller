package main

import (
	"net/http"

	ct "github.com/flynn/flynn-controller/types"
	"github.com/flynn/rpcplus"
	rpc "github.com/flynn/rpcplus/comborpc"
)

func rpcHandler(repo *FormationRepo) http.Handler {
	rpcplus.RegisterName("Controller", &ControllerRPC{formations: repo})
	return rpc.New(rpcplus.DefaultServer)
}

type ControllerRPC struct {
	formations *FormationRepo
}

func (s *ControllerRPC) StreamFormations(arg struct{}, stream rpcplus.Stream) error {
	ch := make(chan *ct.ExpandedFormation)
	if err := s.formations.Subscribe(ch); err != nil {
		return err
	}
	defer func() {
		go func() {
			// drain to prevent deadlock while removing the listener
			for _ = range ch {
			}
		}()
		s.formations.Unsubscribe(ch)
		close(ch)
	}()

	// send sentinel
	select {
	case stream.Send <- &ct.ExpandedFormation{}:
	case <-stream.Error:
		return nil
	}

	for {
		select {
		case f := <-ch:
			select {
			case stream.Send <- f:
			case <-stream.Error:
				return nil
			}
		case <-stream.Error:
			return nil
		}
	}
}
