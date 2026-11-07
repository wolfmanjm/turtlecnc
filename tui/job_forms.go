package tui

type JobField struct {
	label string
	value string
	unit string
}

type JobStruct struct {
	title string
	blurb string
	fields []JobField
}

var Jobs = map[string]JobStruct {
	"Surface": {
		"surface a wasteboard",
		"Starts at current position and z height and mills over the given width, length to given depth",
		[]JobField {
			{"Job Name", "surface", ""},
     		{ "Width", "", "mm" },
     		{ "Length", "", "mm" },
     		{ "Depth", "", "mm" },
     		{ "Tool Diameter", "50", "mm" },
			{ "Feedrate", "1400", "mm/min" },
			{ "Stepover", "40", "%" },
			{ "Spindle",  "", "RPM"},
		},
	},
    "Hole": {
    	"Mill Helical Hole",
    	"Mills a helical hole with an endmill to the given depth",
    	[]JobField {
			{ "Job name", "hole", "" },
			{ "Diameter", "", "mm" },
			{ "Depth", "", "mm" },
			{ "Depth of cut", "0.2", "mm per pass" },
			{ "Tool diameter", "6.35", "mm" },
			{ "Feedrate",  "1400", "mm/min" },
			{ "Peck depth", "0", "mm" },
			{ "Spindle","0", "RPM" },
			{ "Laser", "false", "" },
		},
    },
    "Rectangle": {
	    "Rectangle profile",
	    "Cuts a rectangular profile where Side is inside/outside/laser",
	    []JobField {
	    	{ "Job name", "rectangle", "" },
			{ "Width",    "", ""  },
			{ "Length",   "", ""  },
			{ "Depth",    "", ""  },
			{ "Tool diameter", "6.35", ""  },
			{ "Depth of cut",  "0.2", "mm per pass" },
			{ "Feedrate",   "1400", "mm/min" },
			{ "Spindle",    "", "RPM"},
	        { "Side", "inside|outside|laser", "dropdown"},
	    },
  	},
  	"Circle": {
    	"Circle profile",
    	"Cuts a circlular profile to the given depth, segments is optional",
    	[]JobField {
			{"Job name",      "circle", "" },
			{"Diameter",      "", "" },
			{"Depth",         "", " "},
			{"Tool diameter", "6.35", "" },
			{"Depth of cut",  "0.2", "mm per pass" },
			{"Feedrate",      "1400", "mm/min"},
			{"Spindle",        "", "RPM"},
			{"Segments",       "0", "leave at 0 for auto"},
	        {"Side", "inside|outside|laser", "dropdown"},
	    },
	},

	"Squaring": {
    	"Square some stock",
    	"Square up stock given width and length does top surface, given height will also do sides using a sidecut. depth is final depth of top surface, leave at 0 to just skim by DOC",
	    []JobField {
			{"Job name",      "squaring", "" },
			{"Width",         "", "" },
			{"Length",        "", "" },
			{"Height",        "", "" },
			{"Depth of cut",  "0.1", "mm per pass" },
			{"Depth",         "", "" },
			{"Tool diameter", "12", "" },
			{"Feedrate",      "1400", "mm/min" },
			{"Plungerate",    "100", "mm/min" },
			{"Stepover",      "40", "%" },
			{"Spindle",       "", "RPM"},
		},
	},

  	"Sidemill": {
    	"Side mill an edge",
    	"Starts at current position and z height and side mills over the given width to given depth using specified depth of cut",
	    []JobField {
			{"Job name",      "sidemill", "" },
			{"Length",        "", ""},
			{"Depth",         "", "" },
			{"Depth of cut",  "0.5", "mm per pass" },
			{"Tool diameter", "12", "" },
			{"Feedrate",      "1400", "mm/min"},
			{"Spindle",       "", "RPM"},
			{"Conventional milling", "false", "" },
			{"Side", "bottom|top|left|right", "dropdown"},
    	},
  	},

  	"Pocket": {
    	"Cut a pocket into some stock",
    	"Cut a rectangular pocket into the top of stock",
		[]JobField {
			{"Job name",      "pocket", "" },
			{"Width",         "", "" },
			{"Length",        "", "" },
			{"Depth",         "", "" },
			{"Depth of cut",  "0.1", "mm per pass" },
			{"Tool diameter", "12", "" },
			{"Feedrate",      "1400", "mm/min"},
			{"Plungerate",    "100",  "mm/min"},
			{ "Stepover",     "40",   "%" },
			{"Spindle",       "",     "RPM"},
			{"Spiral path",   "false", ""  },
		},
  },


}



