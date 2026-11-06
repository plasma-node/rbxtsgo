// Package nodepath is a port of Node's path module (lib/path.js, Node 20), for both platforms.
//
// path/filepath is not a substitute. They disagree on inputs the compiler sees: Dir("a/b/") is "a/b" and
// dirname("a/b/") is "a"; Ext(".gitignore") is ".gitignore" and extname(".gitignore") is ""; Rel errors where
// relative returns a path; and Windows comparison rules differ. Byte-exact paths need the same algorithm, so
// this is a line by line port, and the tests compare it with real Node on generated inputs.
//
// Node works on UTF-16 code units and this works on UTF-8 bytes. Separators, dots and colons are ASCII, so the
// results match. The one exception is win32 Relative on a character whose lowercase has a different length,
// such as U+0130: Node's indexes shift there and these don't. No test covers that.
package nodepath

import (
	"os"
	"runtime"
	"strings"
)

// Path is one flavor of Node's path module.
type Path struct {
	Sep   string
	win32 bool
}

var (
	// Win32 is path.win32.
	Win32 = &Path{Sep: `\`, win32: true}
	// Posix is path.posix.
	Posix = &Path{Sep: "/"}
	// Native is require("path") on this platform.
	Native = pickNative()
)

func pickNative() *Path {
	if runtime.GOOS == "windows" {
		return Win32
	}
	return Posix
}

const (
	charSlash     = '/'
	charBackslash = '\\'
	charDot       = '.'
	charColon     = ':'
)

// at is String.charCodeAt, with -1 standing in for NaN.
func at(s string, i int) int {
	if i < 0 || i >= len(s) {
		return -1
	}
	return int(s[i])
}

func isSep(code int) bool      { return code == charSlash || code == charBackslash }
func isPosixSep(code int) bool { return code == charSlash }
func isDeviceRoot(code int) bool {
	return (code >= 'A' && code <= 'Z') || (code >= 'a' && code <= 'z')
}

// normalizeString resolves . and .. elements in a path with directory names.
func normalizeString(path string, allowAboveRoot bool, separator string, isPathSep func(int) bool) string {
	res := ""
	lastSegmentLength := 0
	lastSlash := -1
	dots := 0
	code := 0
	for i := 0; i <= len(path); i++ {
		if i < len(path) {
			code = at(path, i)
		} else if isPathSep(code) {
			break
		} else {
			code = charSlash
		}

		if isPathSep(code) {
			if lastSlash == i-1 || dots == 1 {
				// no-op
			} else if dots == 2 {
				if len(res) < 2 || lastSegmentLength != 2 || at(res, len(res)-1) != charDot || at(res, len(res)-2) != charDot {
					if len(res) > 2 {
						lastSlashIndex := strings.LastIndex(res, separator)
						if lastSlashIndex == -1 {
							res = ""
							lastSegmentLength = 0
						} else {
							res = res[:lastSlashIndex]
							lastSegmentLength = len(res) - 1 - strings.LastIndex(res, separator)
						}
						lastSlash = i
						dots = 0
						continue
					} else if len(res) != 0 {
						res = ""
						lastSegmentLength = 0
						lastSlash = i
						dots = 0
						continue
					}
				}
				if allowAboveRoot {
					if len(res) > 0 {
						res += separator + ".."
					} else {
						res = ".."
					}
					lastSegmentLength = 2
				}
			} else {
				if len(res) > 0 {
					res += separator + path[lastSlash+1:i]
				} else {
					res = path[lastSlash+1 : i]
				}
				lastSegmentLength = i - lastSlash - 1
			}
			lastSlash = i
			dots = 0
		} else if code == charDot && dots != -1 {
			dots++
		} else {
			dots = -1
		}
	}
	return res
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "/"
	}
	return dir
}

func posixCwd() string {
	if runtime.GOOS == "windows" {
		dir := strings.ReplaceAll(cwd(), `\`, "/")
		return dir[strings.Index(dir, "/"):]
	}
	return cwd()
}

// Resolve is path.resolve.
func (p *Path) Resolve(args ...string) string {
	if p.win32 {
		return win32Resolve(args)
	}
	resolvedPath := ""
	resolvedAbsolute := false
	for i := len(args) - 1; i >= -1 && !resolvedAbsolute; i-- {
		var path string
		if i >= 0 {
			path = args[i]
		} else {
			path = posixCwd()
		}
		if len(path) == 0 {
			continue
		}
		resolvedPath = path + "/" + resolvedPath
		resolvedAbsolute = at(path, 0) == charSlash
	}
	resolvedPath = normalizeString(resolvedPath, !resolvedAbsolute, "/", isPosixSep)
	if resolvedAbsolute {
		return "/" + resolvedPath
	}
	if len(resolvedPath) > 0 {
		return resolvedPath
	}
	return "."
}

func win32Resolve(args []string) string {
	resolvedDevice := ""
	resolvedTail := ""
	resolvedAbsolute := false

	for i := len(args) - 1; i >= -1; i-- {
		var path string
		if i >= 0 {
			path = args[i]
			if len(path) == 0 {
				continue
			}
		} else if len(resolvedDevice) == 0 {
			path = cwd()
		} else {
			path = os.Getenv("=" + resolvedDevice)
			if path == "" {
				path = cwd()
			}
			if strings.ToLower(sliceTo(path, 2)) != strings.ToLower(resolvedDevice) && at(path, 2) == charBackslash {
				path = resolvedDevice + `\`
			}
		}

		length := len(path)
		rootEnd := 0
		device := ""
		isAbsolute := false
		code := at(path, 0)

		if length == 1 {
			if isSep(code) {
				rootEnd = 1
				isAbsolute = true
			}
		} else if isSep(code) {
			isAbsolute = true
			if isSep(at(path, 1)) {
				j := 2
				last := j
				for j < length && !isSep(at(path, j)) {
					j++
				}
				if j < length && j != last {
					firstPart := path[last:j]
					last = j
					for j < length && isSep(at(path, j)) {
						j++
					}
					if j < length && j != last {
						last = j
						for j < length && !isSep(at(path, j)) {
							j++
						}
						if j == length || j != last {
							device = `\\` + firstPart + `\` + path[last:j]
							rootEnd = j
						}
					}
				}
			} else {
				rootEnd = 1
			}
		} else if isDeviceRoot(code) && at(path, 1) == charColon {
			device = path[:2]
			rootEnd = 2
			if length > 2 && isSep(at(path, 2)) {
				isAbsolute = true
				rootEnd = 3
			}
		}

		if len(device) > 0 {
			if len(resolvedDevice) > 0 {
				if strings.ToLower(device) != strings.ToLower(resolvedDevice) {
					continue
				}
			} else {
				resolvedDevice = device
			}
		}

		if resolvedAbsolute {
			if len(resolvedDevice) > 0 {
				break
			}
		} else {
			resolvedTail = path[rootEnd:] + `\` + resolvedTail
			resolvedAbsolute = isAbsolute
			if isAbsolute && len(resolvedDevice) > 0 {
				break
			}
		}
	}

	resolvedTail = normalizeString(resolvedTail, !resolvedAbsolute, `\`, isSep)
	if resolvedAbsolute {
		return resolvedDevice + `\` + resolvedTail
	}
	if result := resolvedDevice + resolvedTail; result != "" {
		return result
	}
	return "."
}

func sliceTo(s string, end int) string {
	if end > len(s) {
		return s
	}
	return s[:end]
}

// Normalize is path.normalize.
func (p *Path) Normalize(path string) string {
	if p.win32 {
		return win32Normalize(path)
	}
	if len(path) == 0 {
		return "."
	}
	isAbsolute := at(path, 0) == charSlash
	trailingSeparator := at(path, len(path)-1) == charSlash
	path = normalizeString(path, !isAbsolute, "/", isPosixSep)
	if len(path) == 0 {
		if isAbsolute {
			return "/"
		}
		if trailingSeparator {
			return "./"
		}
		return "."
	}
	if trailingSeparator {
		path += "/"
	}
	if isAbsolute {
		return "/" + path
	}
	return path
}

func win32Normalize(path string) string {
	length := len(path)
	if length == 0 {
		return "."
	}
	rootEnd := 0
	device := ""
	hasDevice := false
	isAbsolute := false
	code := at(path, 0)

	if length == 1 {
		if isPosixSep(code) {
			return `\`
		}
		return path
	}
	if isSep(code) {
		isAbsolute = true
		if isSep(at(path, 1)) {
			j := 2
			last := j
			for j < length && !isSep(at(path, j)) {
				j++
			}
			if j < length && j != last {
				firstPart := path[last:j]
				last = j
				for j < length && isSep(at(path, j)) {
					j++
				}
				if j < length && j != last {
					last = j
					for j < length && !isSep(at(path, j)) {
						j++
					}
					if j == length {
						return `\\` + firstPart + `\` + path[last:] + `\`
					}
					if j != last {
						device = `\\` + firstPart + `\` + path[last:j]
						hasDevice = true
						rootEnd = j
					}
				}
			}
		} else {
			rootEnd = 1
		}
	} else if isDeviceRoot(code) && at(path, 1) == charColon {
		device = path[:2]
		hasDevice = true
		rootEnd = 2
		if length > 2 && isSep(at(path, 2)) {
			isAbsolute = true
			rootEnd = 3
		}
	}

	tail := ""
	if rootEnd < length {
		tail = normalizeString(path[rootEnd:], !isAbsolute, `\`, isSep)
	}
	if len(tail) == 0 && !isAbsolute {
		tail = "."
	}
	if len(tail) > 0 && isSep(at(path, length-1)) {
		tail += `\`
	}
	if !hasDevice {
		if isAbsolute {
			return `\` + tail
		}
		return tail
	}
	if isAbsolute {
		return device + `\` + tail
	}
	return device + tail
}

// IsAbsolute is path.isAbsolute.
func (p *Path) IsAbsolute(path string) bool {
	if !p.win32 {
		return len(path) > 0 && at(path, 0) == charSlash
	}
	if len(path) == 0 {
		return false
	}
	code := at(path, 0)
	return isSep(code) || (len(path) > 2 && isDeviceRoot(code) && at(path, 1) == charColon && isSep(at(path, 2)))
}

// Join is path.join.
func (p *Path) Join(args ...string) string {
	if len(args) == 0 {
		return "."
	}
	joined := ""
	firstPart := ""
	hasJoined := false
	for _, arg := range args {
		if len(arg) > 0 {
			if !hasJoined {
				joined = arg
				firstPart = arg
				hasJoined = true
			} else {
				joined += p.Sep + arg
			}
		}
	}
	if !hasJoined {
		return "."
	}
	if !p.win32 {
		return p.Normalize(joined)
	}

	needsReplace := true
	slashCount := 0
	if isSep(at(firstPart, 0)) {
		slashCount++
		firstLen := len(firstPart)
		if firstLen > 1 && isSep(at(firstPart, 1)) {
			slashCount++
			if firstLen > 2 {
				if isSep(at(firstPart, 2)) {
					slashCount++
				} else {
					needsReplace = false
				}
			}
		}
	}
	if needsReplace {
		for slashCount < len(joined) && isSep(at(joined, slashCount)) {
			slashCount++
		}
		if slashCount >= 2 {
			joined = `\` + joined[slashCount:]
		}
	}
	return win32Normalize(joined)
}

// Relative is path.relative.
func (p *Path) Relative(from, to string) string {
	if from == to {
		return ""
	}
	if p.win32 {
		return win32Relative(from, to)
	}
	from = p.Resolve(from)
	to = p.Resolve(to)
	if from == to {
		return ""
	}
	const fromStart = 1
	fromEnd := len(from)
	fromLen := fromEnd - fromStart
	const toStart = 1
	toLen := len(to) - toStart

	length := min(fromLen, toLen)
	lastCommonSep := -1
	i := 0
	for ; i < length; i++ {
		fromCode := at(from, fromStart+i)
		if fromCode != at(to, toStart+i) {
			break
		} else if fromCode == charSlash {
			lastCommonSep = i
		}
	}
	if i == length {
		if toLen > length {
			if at(to, toStart+i) == charSlash {
				return to[toStart+i+1:]
			}
			if i == 0 {
				return to[toStart+i:]
			}
		} else if fromLen > length {
			if at(from, fromStart+i) == charSlash {
				lastCommonSep = i
			} else if i == 0 {
				lastCommonSep = 0
			}
		}
	}

	out := ""
	for i = fromStart + lastCommonSep + 1; i <= fromEnd; i++ {
		if i == fromEnd || at(from, i) == charSlash {
			if len(out) == 0 {
				out += ".."
			} else {
				out += "/.."
			}
		}
	}
	return out + to[toStart+lastCommonSep:]
}

func win32Relative(from, to string) string {
	fromOrig := win32Resolve([]string{from})
	toOrig := win32Resolve([]string{to})
	if fromOrig == toOrig {
		return ""
	}
	from = lowerSameLength(fromOrig)
	to = lowerSameLength(toOrig)
	if from == to {
		return ""
	}

	fromStart := 0
	for fromStart < len(from) && at(from, fromStart) == charBackslash {
		fromStart++
	}
	fromEnd := len(from)
	for fromEnd-1 > fromStart && at(from, fromEnd-1) == charBackslash {
		fromEnd--
	}
	fromLen := fromEnd - fromStart

	toStart := 0
	for toStart < len(to) && at(to, toStart) == charBackslash {
		toStart++
	}
	toEnd := len(to)
	for toEnd-1 > toStart && at(to, toEnd-1) == charBackslash {
		toEnd--
	}
	toLen := toEnd - toStart

	length := min(fromLen, toLen)
	lastCommonSep := -1
	i := 0
	for ; i < length; i++ {
		fromCode := at(from, fromStart+i)
		if fromCode != at(to, toStart+i) {
			break
		} else if fromCode == charBackslash {
			lastCommonSep = i
		}
	}

	if i != length {
		if lastCommonSep == -1 {
			return toOrig
		}
	} else {
		if toLen > length {
			if at(to, toStart+i) == charBackslash {
				return toOrig[toStart+i+1:]
			}
			if i == 2 {
				return toOrig[toStart+i:]
			}
		}
		if fromLen > length {
			if at(from, fromStart+i) == charBackslash {
				lastCommonSep = i
			} else if i == 2 {
				lastCommonSep = 3
			}
		}
		if lastCommonSep == -1 {
			lastCommonSep = 0
		}
	}

	out := ""
	for i = fromStart + lastCommonSep + 1; i <= fromEnd; i++ {
		if i == fromEnd || at(from, i) == charBackslash {
			if len(out) == 0 {
				out += ".."
			} else {
				out += `\..`
			}
		}
	}

	toStart += lastCommonSep
	if len(out) > 0 {
		return out + toOrig[toStart:toEnd]
	}
	if at(toOrig, toStart) == charBackslash {
		toStart++
	}
	return toOrig[toStart:toEnd]
}

// lowerSameLength lowercases, keeping the byte length, so indexes into the result also fit the original.
func lowerSameLength(s string) string {
	lower := strings.ToLower(s)
	if len(lower) == len(s) {
		return lower
	}
	var b strings.Builder
	for _, r := range s {
		mapped := strings.ToLower(string(r))
		if len(mapped) == len(string(r)) {
			b.WriteString(mapped)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Dirname is path.dirname.
func (p *Path) Dirname(path string) string {
	if p.win32 {
		return win32Dirname(path)
	}
	if len(path) == 0 {
		return "."
	}
	hasRoot := at(path, 0) == charSlash
	end := -1
	matchedSlash := true
	for i := len(path) - 1; i >= 1; i-- {
		if at(path, i) == charSlash {
			if !matchedSlash {
				end = i
				break
			}
		} else {
			matchedSlash = false
		}
	}
	if end == -1 {
		if hasRoot {
			return "/"
		}
		return "."
	}
	if hasRoot && end == 1 {
		return "//"
	}
	return path[:end]
}

func win32Dirname(path string) string {
	length := len(path)
	if length == 0 {
		return "."
	}
	rootEnd := -1
	offset := 0
	code := at(path, 0)

	if length == 1 {
		if isSep(code) {
			return path
		}
		return "."
	}

	if isSep(code) {
		rootEnd = 1
		offset = 1
		if isSep(at(path, 1)) {
			j := 2
			last := j
			for j < length && !isSep(at(path, j)) {
				j++
			}
			if j < length && j != last {
				last = j
				for j < length && isSep(at(path, j)) {
					j++
				}
				if j < length && j != last {
					last = j
					for j < length && !isSep(at(path, j)) {
						j++
					}
					if j == length {
						return path
					}
					if j != last {
						rootEnd = j + 1
						offset = j + 1
					}
				}
			}
		}
	} else if isDeviceRoot(code) && at(path, 1) == charColon {
		if length > 2 && isSep(at(path, 2)) {
			rootEnd = 3
		} else {
			rootEnd = 2
		}
		offset = rootEnd
	}

	end := -1
	matchedSlash := true
	for i := length - 1; i >= offset; i-- {
		if isSep(at(path, i)) {
			if !matchedSlash {
				end = i
				break
			}
		} else {
			matchedSlash = false
		}
	}

	if end == -1 {
		if rootEnd == -1 {
			return "."
		}
		end = rootEnd
	}
	return path[:end]
}

// Basename is path.basename(path, suffix). Pass "" for no suffix: Node treats an empty suffix the same way.
func (p *Path) Basename(path, suffix string) string {
	isPathSep := isPosixSep
	start := 0
	if p.win32 {
		isPathSep = isSep
		if len(path) >= 2 && isDeviceRoot(at(path, 0)) && at(path, 1) == charColon {
			start = 2
		}
	}
	end := -1
	matchedSlash := true

	if len(suffix) > 0 && len(suffix) <= len(path) {
		if suffix == path {
			return ""
		}
		extIdx := len(suffix) - 1
		firstNonSlashEnd := -1
		for i := len(path) - 1; i >= start; i-- {
			code := at(path, i)
			if isPathSep(code) {
				if !matchedSlash {
					start = i + 1
					break
				}
			} else {
				if firstNonSlashEnd == -1 {
					matchedSlash = false
					firstNonSlashEnd = i + 1
				}
				if extIdx >= 0 {
					if code == at(suffix, extIdx) {
						extIdx--
						if extIdx == -1 {
							end = i
						}
					} else {
						extIdx = -1
						end = firstNonSlashEnd
					}
				}
			}
		}
		if start == end {
			end = firstNonSlashEnd
		} else if end == -1 {
			end = len(path)
		}
		return path[start:end]
	}
	for i := len(path) - 1; i >= start; i-- {
		if isPathSep(at(path, i)) {
			if !matchedSlash {
				start = i + 1
				break
			}
		} else if end == -1 {
			matchedSlash = false
			end = i + 1
		}
	}
	if end == -1 {
		return ""
	}
	return path[start:end]
}

// Extname is path.extname.
func (p *Path) Extname(path string) string {
	isPathSep := isPosixSep
	start := 0
	startPart := 0
	if p.win32 {
		isPathSep = isSep
		if len(path) >= 2 && at(path, 1) == charColon && isDeviceRoot(at(path, 0)) {
			start = 2
			startPart = 2
		}
	}
	startDot := -1
	end := -1
	matchedSlash := true
	preDotState := 0

	for i := len(path) - 1; i >= start; i-- {
		code := at(path, i)
		if isPathSep(code) {
			if !matchedSlash {
				startPart = i + 1
				break
			}
			continue
		}
		if end == -1 {
			matchedSlash = false
			end = i + 1
		}
		if code == charDot {
			if startDot == -1 {
				startDot = i
			} else if preDotState != 1 {
				preDotState = 1
			}
		} else if startDot != -1 {
			preDotState = -1
		}
	}

	if startDot == -1 || end == -1 || preDotState == 0 || (preDotState == 1 && startDot == end-1 && startDot == startPart+1) {
		return ""
	}
	return path[startDot:end]
}
