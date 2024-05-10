package vault

import (
	"fmt"
	"testing"
)

func TestParseInitOutput(t *testing.T) {
	expectedUnsealKeys := []string{
		"key1",
		"key2",
		"key3",
		"key4",
		"key5",
	}

	expectedRootToken := "faketokenisfake"

	output := fmt.Sprintf(`{
  "unseal_keys_b64": [
	"%s",
	"%s",
	"%s",
	"%s",
	"%s"
  ],
  "unseal_keys_hex": [
	"hexkey1",
	"hexkey2",
	"hexkey3",
	"hexkey4",
	"hexkey5"
  ],
  "unseal_shares": 5,
  "unseal_threshold": 3,
  "recovery_keys_b64": [],
  "recovery_keys_hex": [],
  "recovery_keys_shares": 0,
  "recovery_keys_threshold": 0,
  "root_token": "%s"
}`,
		expectedUnsealKeys[0],
		expectedUnsealKeys[1],
		expectedUnsealKeys[2],
		expectedUnsealKeys[3],
		expectedUnsealKeys[4],
		expectedRootToken)

	initInfo, err := parseInitOutput(output)

	if err != nil {
		t.Fatalf("parseInitOutput() returned an unexpected error: %v", err)
	}

	if len(initInfo.UnsealKeys) != len(expectedUnsealKeys) {
		t.Fatalf("expected %d unseal keys, got %d", len(expectedUnsealKeys), len(initInfo.UnsealKeys))
	}

	for i, key := range initInfo.UnsealKeys {
		if key != expectedUnsealKeys[i] {
			t.Fatalf("expected unseal key %d to be %s, got %s", i, expectedUnsealKeys[i], key)
		}
	}

	if initInfo.RootToken != expectedRootToken {
		t.Fatalf("expected initial root token to be %s, got %s", expectedRootToken, initInfo.RootToken)
	}
}
