module github.com/oligo/gvcode

go 1.26.1

replace (
	gioui.org v0.9.0 => github.com/ddkwork/gio v0.0.0-20260315032338-a4781d81e6c8
	github.com/go-text/typesetting v0.3.3 => github.com/go-text/typesetting v0.3.0
)

require (
	gioui.org v0.9.0
	github.com/andybalholm/stroke v0.0.0-20251027184313-5126dd7227a1
	github.com/go-text/typesetting v0.3.0
	github.com/rdleal/intervalst v1.5.0
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90
	golang.org/x/image v0.37.0
)

require (
	golang.org/x/exp/shiny v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
)
