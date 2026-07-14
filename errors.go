package tkeyclient

type constError string

func (err constError) Error() string {
	return string(err)
}

func (e constError) Unwrap() error {
	return e
}

