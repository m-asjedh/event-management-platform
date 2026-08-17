package agent

import "fmt"

func dispatch(c *Client, a action) (Observation, error) {
	switch a.tool {
	case "list_events":
		return c.Get("/events?limit=100")
	case "get_event":
		return c.Get("/events/" + a.eventID)
	case "list_sessions":
		return c.Get("/events/" + a.eventID + "/sessions")
	case "list_rooms":
		return c.Get("/events/" + a.eventID + "/rooms")
	case "list_invitations":
		return c.Get("/events/" + a.eventID + "/invitations")
	default:
		return Observation{}, fmt.Errorf("unknown tool %q", a.tool)
	}
}
