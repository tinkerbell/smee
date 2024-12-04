/*
https://github.com/sebest/xff
Copyright (c) 2015 Sebastien Estienne (sebastien.estienne@gmail.com)

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package http

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_none(t *testing.T) {
	res := parse("", nil)
	assert.Equal(t, "", res)
}

func allowAll(string) bool { return true }

func TestParse_localhost(t *testing.T) {
	res := parse("127.0.0.1", allowAll)
	assert.Equal(t, "127.0.0.1", res)
}

func TestParse_invalid(t *testing.T) {
	res := parse("invalid", allowAll)
	assert.Equal(t, "", res)
}

func TestParse_invalid_sioux(t *testing.T) {
	res := parse("123#1#2#3", allowAll)
	assert.Equal(t, "", res)
}

func TestParse_invalid_private_lookalike(t *testing.T) {
	res := parse("102.3.2.1", allowAll)
	assert.Equal(t, "102.3.2.1", res)
}

func TestParse_valid(t *testing.T) {
	res := parse("68.45.152.220", allowAll)
	assert.Equal(t, "68.45.152.220", res)
}

func TestParse_multi_first(t *testing.T) {
	res := parse("12.13.14.15, 68.45.152.220", allowAll)
	assert.Equal(t, "12.13.14.15", res)
}

func TestParse_multi_with_invalid(t *testing.T) {
	res := parse("invalid, 190.57.149.90", allowAll)
	assert.Equal(t, "190.57.149.90", res)
}

func TestParse_multi_with_invalid2(t *testing.T) {
	res := parse("190.57.149.90, invalid", allowAll)
	assert.Equal(t, "", res)
}

func TestParse_multi_with_invalid_sioux(t *testing.T) {
	res := parse("190.57.149.90, 123#1#2#3", allowAll)
	assert.Equal(t, "", res)
}

func TestParse_ipv6_with_port(t *testing.T) {
	res := parse("2604:2000:71a9:bf00:f178:a500:9a2d:670d", allowAll)
	assert.Equal(t, "2604:2000:71a9:bf00:f178:a500:9a2d:670d", res)
}

func TestToMasks_empty(t *testing.T) {
	ips := []string{}
	masks, err := toMasks(ips)
	assert.Empty(t, masks)
	assert.Nil(t, err)
}

func TestToMasks(t *testing.T) {
	ips := []string{"127.0.0.1/32", "10.0.0.0/8"}
	masks, err := toMasks(ips)
	_, ipnet1, _ := net.ParseCIDR("127.0.0.1/32")
	_, ipnet2, _ := net.ParseCIDR("10.0.0.0/8")
	assert.Equal(t, []net.IPNet{*ipnet1, *ipnet2}, masks)
	assert.Nil(t, err)
}

func TestToMasks_error(t *testing.T) {
	ips := []string{"error"}
	masks, err := toMasks(ips)
	assert.Empty(t, masks)
	assert.Equal(t, &net.ParseError{Type: "CIDR address", Text: "error"}, err)
}

func TestAllowed_all(t *testing.T) {
	m, _ := newXFF(xffOptions{
		AllowedSubnets: []string{},
	})
	assert.True(t, m.allowed("127.0.0.1"))
}

func TestAllowed_yes(t *testing.T) {
	m, _ := newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.0/16"},
	})
	assert.True(t, m.allowed("127.0.0.1"))

	m, _ = newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.1/32"},
	})
	assert.True(t, m.allowed("127.0.0.1"))
}

func TestAllowed_no(t *testing.T) {
	m, _ := newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.0/16"},
	})
	assert.False(t, m.allowed("127.1.0.1"))

	m, _ = newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.1/32"},
	})
	assert.False(t, m.allowed("127.0.0.2"))
}

func TestParseUnallowedMidway(t *testing.T) {
	m, _ := newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.0/16"},
	})
	res := parse("1.1.1.1, 8.8.8.8, 127.0.0.1, 127.0.0.2", m.allowed)
	assert.Equal(t, "8.8.8.8", res)
}

func TestParseMany(t *testing.T) {
	m, _ := newXFF(xffOptions{
		AllowedSubnets: []string{"127.0.0.0/16"},
	})
	res := parse("1.1.1.1, 127.0.0.1, 127.0.0.2, 127.0.0.3", m.allowed)
	assert.Equal(t, "1.1.1.1", res)
}
