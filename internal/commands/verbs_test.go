package commands

import "testing"

func TestVerbs_OreAddRegistered(t *testing.T) {
	addCmd, _, err := rootCmd.Find([]string{"add"})
	if err != nil {
		t.Fatalf("'add' command not found: %v", err)
	}
	oreSubCmd, _, err := addCmd.Find([]string{"ore"})
	if err != nil || oreSubCmd == nil {
		t.Errorf("'add ore' verb not registered: %v", err)
	}
}

func TestVerbs_OreGetRegistered(t *testing.T) {
	getCmd, _, err := rootCmd.Find([]string{"get"})
	if err != nil {
		t.Fatalf("'get' command not found: %v", err)
	}
	oreSubCmd, _, err := getCmd.Find([]string{"ore"})
	if err != nil || oreSubCmd == nil {
		t.Errorf("'get ore' verb not registered: %v", err)
	}
}

func TestVerbs_OreNewRegistered(t *testing.T) {
	newCmd, _, err := rootCmd.Find([]string{"new"})
	if err != nil {
		t.Fatalf("'new' command not found: %v", err)
	}
	oreSubCmd, _, err := newCmd.Find([]string{"ore"})
	if err != nil || oreSubCmd == nil {
		t.Errorf("'new ore' verb not registered: %v", err)
	}
}

func TestVerbs_OreRemoveRegistered(t *testing.T) {
	removeCmd, _, err := rootCmd.Find([]string{"remove"})
	if err != nil {
		t.Fatalf("'remove' command not found: %v", err)
	}
	oreSubCmd, _, err := removeCmd.Find([]string{"ore"})
	if err != nil || oreSubCmd == nil {
		t.Errorf("'remove ore' verb not registered: %v", err)
	}
}
