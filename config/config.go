// Copyright (c) 2018-2019 KIDTSUNAMI
// Author: alex@kidtsunami.com
//

// TODO
// - support setting slice values from env variables like PREFIX_VAR_0_VALUE=value

package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const APP_PREFIX = "SPA"

var (
	config   = NewConfig()                  // ReadConfig(), Set()
	defaults = make(map[string]interface{}) // SetDefault()
)

func ConfigName() string {
	return config.ConfigName()
}

func SetConfigName(name string) *Config {
	return config.SetConfigName(name)
}

func UseEnv(enabled bool) *Config {
	return config.UseEnv(enabled)
}

func SetEnvPrefix(p string) *Config {
	return config.SetEnvPrefix(p)
}

func ReadConfigFile() error {
	return config.ReadConfigFile()
}

func ReadConfig(buf []byte) error {
	return config.ReadConfig(buf)
}

func Set(key string, val interface{}) *Config {
	return config.Set(key, val)
}

func SetDefault(key string, val interface{}) *Config {
	return config.SetDefault(key, val)
}

func GetString(path string) string {
	return config.GetString(path)
}

func GetStringSlice(path string) []string {
	return config.GetStringSlice(path)
}

func GetStringMap(path string) map[string]string {
	return config.GetStringMap(path)
}

func GetDuration(path string) time.Duration {
	return config.GetDuration(path)
}

func GetBool(path string) bool {
	return config.GetBool(path)
}

func GetInt64(path string) int64 {
	return config.GetInt64(path)
}

func GetUint64(path string) uint64 {
	return config.GetUint64(path)
}

func GetInt64Slice(path string) []int64 {
	return config.GetInt64Slice(path)
}

func GetUint64Slice(path string) []uint64 {
	return config.GetUint64Slice(path)
}

func GetInt(path string) int {
	return config.GetInt(path)
}

func GetUint(path string) uint {
	return config.GetUint(path)
}

func GetIntSlice(path string) []int {
	return config.GetIntSlice(path)
}

func GetUintSlice(path string) []uint {
	return config.GetUintSlice(path)
}

func GetFloat64(path string) float64 {
	return config.GetFloat64(path)
}

func GetFloat64Slice(path string) []float64 {
	return config.GetFloat64Slice(path)
}

func AllSettings() map[string]interface{} {
	return config.AllSettings()
}

func Unmarshal(path string, val interface{}) error {
	return config.Unmarshal(path, val)
}

func ForEach(path string, fn func(c *Config) error) error {
	return config.ForEach(path, fn)
}

type Config struct {
	confName  string
	envPrefix string
	noEnv     bool
	data      map[string]interface{} // read from config file or set
	merged    map[string]interface{} // merged env, data, defaults
}

func NewConfig() *Config {
	return &Config{
		data:   make(map[string]interface{}),
		merged: nil,
	}
}

func canAccess(name string) bool {
	s, err := os.Stat(name)
	return err == nil && !s.IsDir()
}

func (c *Config) ConfigName() string {
	name := c.confName
	if name == "" || !canAccess(name) {
		name = os.Getenv(c.expandEnvKey("CONFIG_FILE"))
	}
	if name == "" || !canAccess(name) {
		name = "config.json"
	}
	return name
}

func (c *Config) SetConfigName(name string) *Config {
	c.confName = name
	return c
}

func (c *Config) SetEnvPrefix(p string) *Config {
	c.envPrefix = strings.ToUpper(p)
	c.merged = nil
	return c
}

func (c *Config) UseEnv(enabled bool) *Config {
	c.noEnv = !enabled
	return c
}

func (c *Config) ReadConfigFile() error {
	// determine config name from
	// - local variable
	// - environment
	// - fallback: use config.json
	name := c.ConfigName()

	// read config file
	buf, err := ioutil.ReadFile(name)
	if err != nil {
		return fmt.Errorf("reading config file: %v", err)
	}
	return c.ReadConfig(buf)
}

func (c *Config) ReadConfig(buf []byte) error {
	// unpack config from JSON into Go map
	if err := json.Unmarshal(buf, &c.data); err != nil {
		return fmt.Errorf("parsing config file: %v", err)
	}
	c.merged = nil
	// parse env for any defined value
	_ = c.AllSettings()
	return nil
}

func (c *Config) expandEnvKey(key string) string {
	if c.envPrefix != "" {
		return c.envPrefix + "_" + key
	}
	return key
}

func (c *Config) Set(key string, val interface{}) *Config {
	setTree(c.data, key, val)
	c.merged = nil
	return c
}

func (c *Config) SetDefault(key string, val interface{}) *Config {
	setTree(defaults, key, val)
	c.merged = nil
	return c
}

func setTree(walker map[string]interface{}, key string, val interface{}) {
	keys := strings.Split(key, ".")
	for n, v := range keys {
		if sub, ok := walker[v]; ok {
			if submap, ok := sub.(map[string]interface{}); ok {
				walker = submap
			} else if n == len(keys)-1 {
				walker[v] = val
			} else {
				log.Fatalf("config: cannot set path '%s': %s exists as type %T", key, v, sub)
			}
		} else if n < len(keys)-1 {
			// append subtree
			sub := make(map[string]interface{})
			walker[v] = sub
			walker = sub
		} else {
			// append leaf
			walker[v] = val
		}
	}
}

func getTree(walker map[string]interface{}, key string) interface{} {
	keys := strings.Split(key, ".")
	for n, v := range keys {
		if sub, ok := walker[v]; ok {
			if n == len(keys)-1 {
				return sub
			}
			if submap, ok := sub.(map[string]interface{}); ok {
				walker = submap
			} else {
				break
			}
		}
	}
	return nil
}

func (c *Config) getEnv(path string) (string, bool) {
	if c.noEnv {
		return "", false
	}
	path = strings.ToUpper(path)
	path = strings.Replace(path, ".", "_", -1)
	return os.LookupEnv(c.expandEnvKey(path))
}

func (c *Config) getValue(path string) interface{} {
	// get env key when present (allows to overwrite with empty value)
	if val, ok := c.getEnv(path); ok {
		return val
	}
	// try get merged tree (env + config + defaults)
	if c.merged != nil {
		if val := getTree(c.merged, path); val != nil {
			return val
		}
	}
	// get config file data if set
	if val := getTree(c.data, path); val != nil {
		return val
	}
	// get default if registered
	if val := getTree(defaults, path); val != nil {
		return val
	}
	return nil
}

func (c *Config) GetString(path string) string {
	val := c.getValue(path)
	if val == nil {
		return ""
	}
	return toString(val)
}

func (c *Config) GetStringSlice(path string) []string {
	val := c.getValue(path)
	if val == nil {
		return []string{}
	}
	if stringslice, ok := val.([]string); ok {
		return stringslice
	}
	if vslice, ok := val.([]interface{}); ok {
		s := make([]string, len(vslice))
		for i, v := range vslice {
			s[i] = toString(v)
		}
		return s
	}
	return []string{toString(val)}
}

func (c *Config) GetStringMap(path string) map[string]string {
	val := c.getValue(path)
	smap := make(map[string]string)
	if val == nil {
		return smap
	}
	if m, ok := val.(map[string]interface{}); ok {
		for n, v := range m {
			if s := toString(v); s != "" {
				smap[n] = s
			}
		}
	}
	return smap
}

func (c *Config) GetDuration(path string) time.Duration {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case time.Duration:
		return v
	case int:
		return time.Duration(v)
	case int32:
		return time.Duration(v)
	case uint32:
		return time.Duration(v)
	case int64:
		return time.Duration(v)
	case uint64:
		return time.Duration(v)
	case string:
		if dur, err := parseDuration(v); err == nil {
			return dur
		}
	default:
		s := toString(v)
		if dur, err := parseDuration(s); err == nil {
			return dur
		}
	}
	return 0
}

func (c *Config) GetBool(path string) bool {
	val := c.getValue(path)
	if val == nil {
		return false
	}
	switch v := val.(type) {
	case bool:
		return v
	default:
		s := toString(v)
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	}
	return false
}

func (c *Config) GetInt(path string) int {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	default:
		s := toString(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return int(i)
		}
	}
	return 0
}

func (c *Config) GetUint(path string) uint {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case uint64:
		return uint(v)
	case float64:
		return uint(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return uint(n)
		}
	default:
		s := toString(v)
		if i, err := strconv.ParseUint(s, 10, 64); err == nil {
			return uint(i)
		}
	}
	return 0
}

func (c *Config) GetIntSlice(path string) []int {
	is := make([]int, 0)
	for _, v := range c.GetStringSlice(path) {
		val, _ := strconv.Atoi(v)
		is = append(is, val)
	}
	return is
}

func (c *Config) GetUintSlice(path string) []uint {
	is := make([]uint, 0)
	for _, v := range c.GetStringSlice(path) {
		val, _ := strconv.Atoi(v)
		is = append(is, uint(val))
	}
	return is
}

func (c *Config) GetInt64(path string) int64 {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
	default:
		s := toString(v)
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func (c *Config) GetUint64(path string) uint64 {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case uint64:
		return v
	case float64:
		return uint64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return uint64(n)
		}
	default:
		s := toString(v)
		if i, err := strconv.ParseUint(s, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

func (c *Config) GetInt64Slice(path string) []int64 {
	is := make([]int64, 0)
	for _, v := range c.GetStringSlice(path) {
		val, _ := strconv.ParseInt(v, 10, 64)
		is = append(is, val)
	}
	return is
}

func (c *Config) GetUint64Slice(path string) []uint64 {
	is := make([]uint64, 0)
	for _, v := range c.GetStringSlice(path) {
		val, _ := strconv.ParseUint(v, 10, 64)
		is = append(is, val)
	}
	return is
}

func (c *Config) GetFloat64(path string) float64 {
	val := c.getValue(path)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int64:
		return float64(v)
	case float64:
		return v
	case json.Number:
		if n, err := v.Float64(); err == nil {
			return n
		}
	default:
		s := toString(v)
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return 0
}

func (c *Config) GetFloat64Slice(path string) []float64 {
	is := make([]float64, 0)
	for _, v := range c.GetStringSlice(path) {
		val, _ := strconv.ParseFloat(v, 64)
		is = append(is, val)
	}
	return is
}

func (c *Config) AllSettings() map[string]interface{} {
	if c.merged != nil {
		return c.merged
	}
	c.merged = make(map[string]interface{})

	// copy map pcontents (we use JSON marshaling to simplify this code)
	buf, _ := json.Marshal(&defaults)
	json.Unmarshal(buf, &c.merged)

	// merge data map into defaults (overwrites only values defined in data)
	buf, _ = json.Marshal(&c.data)
	json.Unmarshal(buf, &c.merged)

	// extend keys with matching env variables, only if env prefix is set
	if !c.noEnv && c.envPrefix == "" {
		return c.merged
	}

	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, c.envPrefix) {
			continue
		}
		fields := strings.SplitN(v, "=", 2)
		key := strings.Join(strings.SplitN(strings.ToLower(fields[0]), "_", 3)[1:], ".")
		setTree(c.merged, key, fields[1])
	}
	return c.merged
}

func (c *Config) ForEach(path string, fn func(c *Config) error) error {
	// requires merged tree
	s := c.AllSettings()
	segs := strings.Split(path, ".")
	var slice []interface{}
	for i, v := range segs[:len(segs)-1] {
		if s == nil {
			break
		}
		sub, ok := s[v]
		if !ok {
			return fmt.Errorf("missing config path '%s'", path)
		}
		s, ok = sub.(map[string]interface{})
		if !ok && i < len(segs)-1 {
			return fmt.Errorf("invalid type %T at config path '%s'", sub, path)
		}
	}
	// assuming the last sub-tree element is a slice
	slice, ok := s[segs[len(segs)-1]].([]interface{})
	if !ok {
		return fmt.Errorf("expected slice of values at path '%s'", path)
	}
	for _, v := range slice {
		err := fn(&Config{
			noEnv:  true,
			data:   v.(map[string]interface{}),
			merged: v.(map[string]interface{}),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) Unmarshal(path string, val interface{}) error {
	// requires merged tree
	s := c.AllSettings()
	for _, v := range strings.Split(path, ".") {
		if s == nil {
			break
		}
		sub, ok := s[v]
		if !ok {
			return fmt.Errorf("missing config path '%s'", path)
		}
		s, ok = sub.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid type %T at config path '%s'", sub, path)
		}
	}
	buf, _ := json.Marshal(s)
	return json.Unmarshal(buf, val)
}

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

func toString(t interface{}) string {
	val := reflect.Indirect(reflect.ValueOf(t))
	if !val.IsValid() {
		return ""
	}
	if val.Type().Implements(stringerType) {
		return t.(fmt.Stringer).String()
	}
	if s, err := toRawString(val.Interface()); err == nil {
		return s
	}
	return fmt.Sprintf("%v", val.Interface())
}

func toRawString(t interface{}) (string, error) {
	val := reflect.Indirect(reflect.ValueOf(t))
	if !val.IsValid() {
		return "", nil
	}
	typ := val.Type()
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, val.Type().Bits()), nil
	case reflect.String:
		return val.String(), nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil
	case reflect.Array:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// [...]byte
		var b []byte
		if val.CanAddr() {
			b = val.Slice(0, val.Len()).Bytes()
		} else {
			b = make([]byte, val.Len())
			reflect.Copy(reflect.ValueOf(b), val)
		}
		return hex.EncodeToString(b), nil
	case reflect.Slice:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// []byte
		b := val.Bytes()
		return hex.EncodeToString(b), nil
	}
	return "", fmt.Errorf("no method for converting type %s (%v) to string", typ.String(), val.Kind())
}

func parseDuration(d string) (time.Duration, error) {
	// parse integer values as seconds
	if i, err := strconv.ParseInt(d, 10, 64); err == nil {
		return time.Duration(i) * time.Second, nil
	}
	// parse as duration string (note: no whitespace allowed)
	if i, err := time.ParseDuration(d); err == nil {
		return i, nil
	}
	// parse as duration string with whitespace removed
	d = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, d)
	if i, err := time.ParseDuration(d); err == nil {
		return i, nil
	}
	return 0, fmt.Errorf("parsing duration \"%s\": invalid syntax", d)
}
