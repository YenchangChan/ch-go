package proto

import "github.com/go-faster/errors"

type Query struct {
	ID          string
	Body        string
	Secret      string
	Stage       Stage
	Compression Compression
	Info        ClientInfo
	Settings    []Setting
}

type Setting struct {
	Key       string
	Value     string
	Important bool
}

func (s Setting) Encode(b *Buffer) {
	b.PutString(s.Key)
	b.PutBool(s.Important)
	b.PutString(s.Value)
}

func (s *Setting) Decode(r *Reader) error {
	key, err := r.Str()
	if err != nil {
		return errors.Wrap(err, "key")
	}

	if key == "" {
		// End of settings.
		return nil
	}

	important, err := r.Bool()
	if err != nil {
		return errors.Wrap(err, "important")
	}
	v, err := r.Str()
	if err != nil {
		return errors.Wrap(err, "value")
	}

	s.Key = key
	s.Important = important
	s.Value = v

	return nil
}

func (q *Query) DecodeAware(r *Reader, revision int) error {
	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "query id")
		}
		q.ID = v
	}
	if FeatureClientWriteInfo.In(revision) {
		if err := q.Info.DecodeAware(r, revision); err != nil {
			return errors.Wrap(err, "client info")
		}
	}
	if !FeatureSettingsSerializedAsStrings.In(revision) {
		return errors.New("unsupported revision")
	}
	for {
		var s Setting
		if err := s.Decode(r); err != nil {
			return errors.Wrap(err, "setting")
		}
		if s.Key == "" {
			break
		}
		q.Settings = append(q.Settings, s)
	}
	if FeatureInterServerSecret.In(revision) {
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "inter-server secret")
		}
		q.Secret = v
	}

	{
		v, err := r.Uvarint()
		if err != nil {
			return errors.Wrap(err, "stage")
		}
		q.Stage = Stage(v)
		if !q.Stage.IsAStage() {
			return errors.Errorf("unknown stage %d", v)
		}
	}
	{
		v, err := r.Uvarint()
		if err != nil {
			return errors.Wrap(err, "stage")
		}
		q.Compression = Compression(v)
		if !q.Compression.IsACompression() {
			return errors.Errorf("unknown compression %d", v)
		}
	}

	{
		v, err := r.Str()
		if err != nil {
			return errors.Wrap(err, "query body")
		}
		q.Body = v
	}

	return nil
}

func (q Query) EncodeAware(b *Buffer, revision int) {
	ClientCodeQuery.Encode(b)
	b.PutString(q.ID)
	if FeatureClientWriteInfo.In(revision) {
		q.Info.EncodeAware(b, revision)
	}
	if FeatureSettingsSerializedAsStrings.In(revision) {
		for _, s := range q.Settings {
			s.Encode(b)
		}
	}
	b.PutString("")

	if FeatureInterServerSecret.In(revision) {
		b.PutString(q.Secret)
	}

	StageComplete.Encode(b)
	q.Compression.Encode(b)

	b.PutString(q.Body)
}