package platform

// Input and output shapes the module handlers share.

type IDInput struct {
	ID int64 `path:"id"`
}

// VersionInput names a record and the version the caller last read, for a delete.
type VersionInput struct {
	ID      int64 `path:"id"`
	Version int32 `query:"version" required:"true"`
}

type CreatedOutput struct {
	Body struct {
		ID int64 `json:"id"`
	}
}

// Created answers a create with the new id, or its error.
func Created(id int64, err error) (*CreatedOutput, error) {
	if err != nil {
		return nil, err
	}
	out := &CreatedOutput{}
	out.Body.ID = id
	return out, nil
}

// ActionsOutput lists what the actor may do before picking a record.
type ActionsOutput struct {
	Body struct {
		AllowedActions []string `json:"allowed_actions" nullable:"false"`
	}
}
