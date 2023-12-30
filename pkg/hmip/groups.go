package hmip

import "encoding/json"

type group struct {
	stateful
	named
	typed
}

// ======================================================

type metaGroup struct {
	group
	Icon string `json:"groupIcon"`
}

func (mg metaGroup) GetIcon() string {
	return mg.Icon
}

// ======================================================

func (g *Groups) UnmarshalJSON(value []byte) error {
	var groupValues map[string]json.RawMessage
	err := json.Unmarshal(value, &groupValues)
	if err != nil {
		return err
	}
	groups := make(Groups, 0, len(groupValues))
	for _, groupValue := range groupValues {
		var group group
		err = json.Unmarshal(groupValue, &group)
		if err != nil {
			return err
		}
		switch group.Type {
		case GROUP_TYPE_META:
			specialGroup := metaGroup{
				group: group,
			}
			err = json.Unmarshal(groupValue, &specialGroup)
			if err != nil {
				return err
			}
			groups = append(groups, &specialGroup)
		default:
			groups = append(groups, &group)
		}
	}
	*g = groups
	return nil
}
