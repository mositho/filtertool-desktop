package compiler

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type rule struct {
	lines  []string
	hidden bool
}

var operator = regexp.MustCompile(`^(<=|>=|==|!=|<|>|=)`)

func newRule() *rule { return &rule{lines: []string{}} }
func (r *rule) set(key, value string) *rule {
	if strings.TrimSpace(value) == "" {
		return r
	}
	op := operator.FindString(value)
	out := []string{}
	for _, line := range r.lines {
		if strings.HasPrefix(line, key+" ") && (op == "" || operator.FindString(strings.TrimPrefix(line, key+" ")) == op) {
			continue
		}
		out = append(out, line)
	}
	r.lines = append(out, key+" "+value)
	return r
}
func (r *rule) number(key, op string, n any) *rule {
	if n != nil {
		r.set(key, op+" "+fmt.Sprint(n))
	}
	return r
}
func (r *rule) quoted(key string, exact bool, values ...string) *rule {
	if len(values) == 0 {
		return r
	}
	a := []string{}
	for _, v := range values {
		a = append(a, strconv.Quote(v))
	}
	s := strings.Join(a, " ")
	if exact {
		s = "== " + s
	}
	return r.set(key, s)
}
func (r *rule) icon(color, shape string, size any) *rule {
	return r.set("MinimapIcon", fmt.Sprintf("%v %s %s", size, color, shape))
}
func (r *rule) hide() *rule {
	r.hidden = true
	out := []string{}
	for _, s := range r.lines {
		if strings.HasPrefix(s, "MinimapIcon ") || strings.HasPrefix(s, "PlayAlertSound") || strings.HasPrefix(s, "CustomAlertSound") || strings.HasPrefix(s, "PlayEffect ") {
			continue
		}
		out = append(out, s)
	}
	r.lines = out
	return r
}
func (r *rule) render() string {
	kind := "Show"
	if r.hidden {
		kind = "Hide"
	}
	return kind + "\n" + strings.Join(r.lines, "\n")
}
func rgb(hex string) string {
	v := strings.TrimPrefix(hex, "#")
	if len(v) == 3 {
		v = string([]byte{v[0], v[0], v[1], v[1], v[2], v[2]})
	}
	if len(v) != 6 {
		return "0 0 0"
	}
	n, err := strconv.ParseUint(v, 16, 32)
	if err != nil {
		return "0 0 0"
	}
	return fmt.Sprintf("%d %d %d", n>>16, (n>>8)&255, n&255)
}
func (r *rule) style(s Object) *rule {
	for _, pair := range [][3]string{{"text", "SetTextColor", "#C8C8C8"}, {"background", "SetBackgroundColor", "#000000"}, {"border", "SetBorderColor", "#000000"}} {
		v, exists := s[pair[0]]
		if exists && v == nil {
			continue
		}
		color := pair[2]
		if exists {
			color = Text(v)
		}
		value := rgb(color)
		if pair[0] == "background" {
			if opacity, ok := s["backgroundOpacity"].(float64); ok {
				value += fmt.Sprintf(" %.0f", math.Round(opacity*255))
			}
		}
		r.set(pair[1], value)
	}
	return r.set("SetFontSize", fmt.Sprint(math.Max(20, Num(s["size"], 45))))
}
func heading(name string, rules ...*rule) string {
	parts := []string{"### " + name}
	for _, r := range rules {
		if r != nil {
			parts = append(parts, r.render())
		}
	}
	return strings.Join(parts, "\n\n")
}
func (c *Compiler) styled(name string) *rule { return newRule().style(Obj(c.Styles[name])) }
func (c *Compiler) soundPath(name string) string {
	if strings.HasPrefix(name, c.SoundFolder+"/") {
		return name
	}
	return c.SoundFolder + "/" + name
}
func (c *Compiler) tts(r *rule, value string) *rule {
	name := strings.ReplaceAll(value, " ", "_")
	if _, ok := c.manifest[value]; ok {
		name = value
	}
	return r.set("CustomAlertSound", strconv.Quote(c.soundPath(name+".mp3"))+" 300")
}
func (c *Compiler) sound(r *rule, s Object) *rule {
	if s["enabled"] == false {
		return r
	}
	if t := Text(s["tts"]); t != "" {
		return c.tts(r, t)
	}
	if f := Text(s["soundFileName"]); f != "" {
		return r.set("CustomAlertSound", strconv.Quote(c.soundPath(f))+" 300")
	}
	if id, ok := s["soundId"]; ok {
		return r.set("PlayAlertSound", fmt.Sprint(id)+" 240")
	}
	return r
}
func (c *Compiler) presentation(r *rule, cfg, base Object) *rule {
	s := Obj(cfg["style"])
	style := Merge(base, Obj(c.Styles[Text(s["preset"])]))
	for k, v := range s {
		if k != "preset" {
			style[k] = v
		}
	}
	r.style(style)
	i := Obj(cfg["icon"])
	if i["enabled"] != false && (Text(i["iconColor"]) != "" || Text(i["iconShape"]) != "") {
		r.icon(Text(Val(i, "iconColor", "Cyan")), Text(Val(i, "iconShape", "Circle")), Val(i, "iconSize", float64(2)))
	}
	return c.sound(r, Obj(cfg["sound"]))
}
func defence(r *rule, kind string) *rule {
	a, e, s := "==", "==", "=="
	if strings.Contains(kind, "armour") {
		a = ">="
	}
	if strings.Contains(kind, "evasion") {
		e = ">="
	}
	if strings.Contains(kind, "es") {
		s = ">="
	}
	n := func(op string) string {
		if op == ">=" {
			return "1"
		}
		return "0"
	}
	return r.set("BaseArmour", a+" "+n(a)).set("BaseEnergyShield", s+" "+n(s)).set("BaseEvasion", e+" "+n(e))
}
