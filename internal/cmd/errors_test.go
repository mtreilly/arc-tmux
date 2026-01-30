package cmd

import "testing"

func TestCodedErrorFormatting(t *testing.T) {
	err := newCodedError("ERR_TEST", "something went wrong", nil)
	if err.Error() != "ERR_TEST: something went wrong" {
		t.Fatalf("unexpected error string: %s", err.Error())
	}

	cause := newCodedError("ERR_CAUSE", "bad", nil)
	err = newCodedError("ERR_TEST", "something went wrong", cause)
	expected := "ERR_TEST: something went wrong: ERR_CAUSE: bad"
	if err.Error() != expected {
		t.Fatalf("unexpected error string: %s", err.Error())
	}
}
