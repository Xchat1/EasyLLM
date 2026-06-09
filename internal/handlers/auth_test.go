package handlers

import "testing"

func TestPasswordMeetsMinimum(t *testing.T) {
	if passwordMeetsMinimum("1234567") {
		t.Fatalf("expected seven-character password to be rejected")
	}
	if !passwordMeetsMinimum("12345678") {
		t.Fatalf("expected eight-character password to be accepted")
	}
	if passwordMeetsMinimum("密码密码密码") {
		t.Fatalf("expected six-rune password to be rejected")
	}
}
