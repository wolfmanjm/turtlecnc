package tui

type JobField struct {
	label string
	value string
	unit  string
}

type JobStruct struct {
	title  string
	blurb  string
	fields []JobField
}

var Jobs = map[string]JobStruct{
	"Surface": {
		"surface a wasteboard",
		"Starts at current position (which should be front left of surface) and z height and mills over the given width, length to given depth",
		[]JobField{
			{"Job name", "surface", "Set filename to this (.nc)"},
			{"Width", "", "(x) in mm"},
			{"Length", "", "(y) in mm"},
			{"Depth", "", "(-z) in mm"},
			{"Tool Diameter", "50", "in mm"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Stepover", "40", "stepover as percentage of tool diameter"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
		},
	},
	"Hole": {
		"Mill Helical Hole",
		"Mills a helical hole with an endmill to the given depth, center of hole is at current position",
		[]JobField{
			{"Job name", "hole", "Set filename to this (.nc)"},
			{"Diameter", "", "of hole in mm"},
			{"Depth", "", "(-z) in mm"},
			{"Depth of cut", "0.2", "in mm per pass"},
			{"Tool diameter", "6.35", "in mm"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Peck depth", "0", "if set will withdraw (+z) every set amount in mm"},
			{"Spindle", "0", "If specified turn on spindle at RPM"},
			{"Laser", "false", "if checked will produce gcode for a laser"},
		},
	},
	"Rectangle": {
		"Rectangle profile",
		"Cuts a rectangular profile where Side is inside/outside/laser, bottom left is at current position",
		[]JobField{
			{"Job name", "rectangle", "Set filename to this (.nc)"},
			{"Width", "", "(x) in mm"},
			{"Length", "", "(y) in mm"},
			{"Depth", "", "(-z) in mm"},
			{"Tool diameter", "6.35", "in mm"},
			{"Depth of cut", "0.2", "in mm per pass"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
			{"Side", "inside|outside|laser", "Select which side the cut is on (laser will be on the line)"},
		},
	},
	"Circle": {
		"Circle profile",
		"Cuts a circlular profile to the given depth, segments is optional",
		[]JobField{
			{"Job name", "circle", "Set filename to this (.nc)"},
			{"Diameter", "", "of circle in mm"},
			{"Depth", "", "(-z) in mm"},
			{"Tool diameter", "6.35", "in mm"},
			{"Depth of cut", "0.2", "in mm per pass"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
			{"Segments", "0", "number of segments in circle, 0 for auto, 6 for hex"},
			{"Side", "inside|outside|laser", "Select which side the cut is on (laser will be on the line)"},
		},
	},

	"Squaring": {
		"Square some stock",
		"Square up stock given width and length does top surface, given height will also do sides using a sidecut. depth is final depth of top surface, leave at 0 to just skim by DOC. WCS X0 Y0 needs to be set at front left corner.",
		[]JobField{
			{"Job name", "squaring", "Set filename to this (.nc)"},
			{"Width", "", "(x) in mm"},
			{"Length", "", "(y) in mm"},
			{"Height", "", "if set will also sidemill around at this height (-z) in mm"},
			{"Depth of cut", "0.1", "in mm per pass"},
			{"Depth", "", "final depth (-z) of top surface in mm"},
			{"Tool diameter", "12", "in mm"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Plungerate", "100", "cutting plunge rate in mm/min"},
			{"Stepover", "40", "stepover as percentage of tool diameter"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
		},
	},

	"Sidemill": {
		"Side mill an edge",
		"Starts at current position and z height and side mills over the given width to given depth using specified depth of cut",
		[]JobField{
			{"Job name", "sidemill", "Set filename to this (.nc)"},
			{"Length", "", "(y) in mm"},
			{"Depth", "", "amount to cut off side in mm"},
			{"Depth of cut", "0.5", "in mm per pass"},
			{"Tool diameter", "12", "in mm"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
			{"Conventional milling", "false", "if not checked will use climb"},
			{"Side", "bottom|top|left|right", "selects which side to mill"},
		},
	},

	"Pocket": {
		"Cut a pocket into some stock",
		"Cut a rectangular pocket into the top of stock. WCS X0 Y0 needs to be set at front left corner of the pocket",
		[]JobField{
			{"Job name", "pocket", "Set filename to this (.nc)"},
			{"Width", "", "(x) in mm"},
			{"Length", "", "(y) in mm"},
			{"Depth", "", "(-z) in mm"},
			{"Depth of cut", "0.1", "in mm per pass"},
			{"Tool diameter", "12", "in mm"},
			{"Feedrate", "1400", "cutting feedrate in mm/min"},
			{"Plungerate", "100", "cutting plunge rate in mm/min"},
			{"Stepover", "40", "stepover as percentage of tool diameter"},
			{"Spindle", "", "If specified turn on spindle at RPM"},
			{"Spiral path", "false", "if checked will use a spiral cutting path"},
		},
	},
}
