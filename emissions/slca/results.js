var lcaData;
var map;
var currentView = "";
var iconHeight = 35; // height of sidebar toggle icon


function getState() {
  var sel1 = document.getElementById("pathselect");
  var sel2 = document.getElementById("resultVarSelect");
  var sel3 = document.getElementById("viewTypeSelect");
  var unitsdiv = document.getElementById("units");
  var state = {
    pathID: sel1.options[sel1.selectedIndex].value,
    amt: document.getElementById("amt").value,
    units: unitsdiv.options[unitsdiv.selectedIndex].value,
    resultVar: sel2.options[sel2.selectedIndex].value,
    viewType: sel3.options[sel3.selectedIndex].value,
    windowHeight: $(window).height()
  };
  return state;
}

var updateMajor = function() {
  s = getState();

  switch (s.viewType) {
    case "network":
      s.currentView = "network";
      newNetwork(s);
      break;
    case "map":
      if (s.currentView == "map") {
        updateMap(s);
      } else {
        s.currentView = "map";
        newMap(s);
      }
      break;
    default:
      console.log("Unknown view type " + s.viewType);
      break;
  }
};

var updateMinor = function() {
  s = getState();

  switch (s.viewType) {
    case "network":
      s.currentView = "network";
      updateNetwork(s);
      break;
    case "map":
      if (s.currentView == "map") {
        updateMap(s);
      } else {
        s.currentView = "map";
        newMap(s);
      }
      break;
    default:
      console.log("Unknown view type " + s.viewType);
      break;
  }
};

var nodeColor = function(val, isExpanded, procType) {
  if (val < 0) {
    color = "#abd8ff";
  } else if (val == 0) {
    color = "#ffffff";
  } else {
    color = "#ffabab";
  }
  return color;
};

var newNetwork = function(s) {
  $.getJSON("{{.}}/results?pathselect=" + s.pathID + "&amt=" + s.amt + "&units=" + s.units, function(result) {
    lcaData = result;
    updateResultOptions();
    updateNetwork(s);
  });
};

var updateResultOptions = function() {
  $(function() {
    var $select = $('#resultVarSelect');
    $select.empty();

    var group = $('<optgroup label="Emissions" />');
    $.each(lcaData.Emissions, function() {
      label = this.Name + " (" + this.Value.toPrecision(2) + " " + this.Units + ")";
      $('<option />').html(label).prop("value", this.ID).appendTo(group);
    });
    group.appendTo($select);

    var group = $('<optgroup label="Resources" />');
    $.each(lcaData.Resources, function() {
      label = this.Name + " (" + this.Value.toPrecision(2) + " " + this.Units + ")";
      $('<option />').html(label).prop("value", this.ID).appendTo(group);
    });
    group.appendTo($select);
  });
}

var nodeLabel = function(name, amt, units) {
  return name + "\n" + amt.toPrecision(2) + " " + s.units
}

var amtUnits = function(results, varname) {
  var amt = 0;
  var units = "";
  if (varname in results) {
    amt = results[varname].Value;
    units = results[varname].Units;
  }
  return [amt, units];
}

var borderWidth = function(procType) {
  if (procType == "Mix") {
    return 1;
  }
  return 3;
}

var updateNetwork = function(s) {
  var nodes = new vis.DataSet();
  var edges = new vis.DataSet();
  for (var i in lcaData.Nodes) {
    var node = lcaData.Nodes[i];
    var amtunits = amtUnits(node.Results, s.resultVar);
    var amt = amtunits[0];
    var units = amtunits[1];
    nodes.add({
      id: node.ID,
      pathName: node.PathName,
      label: nodeLabel(node.ProcName, amt, units),
      value: amt,
      color: nodeColor(amt, true, node.ProcType)
        //borderWidth: borderWidth(node.ProcType)
    });
  }
  for (var i in lcaData.Edges) {
    var edge = lcaData.Edges[i];
    var amtunits = amtUnits(edge.Results, s.resultVar);
    edges.add({
      from: edge.FromID,
      to: edge.ToID,
      value: amtunits[0]
    });
  }
  // create a network
  var container = document.getElementById('viewer');
  container.style.backgroundColor = 'white';
  var data = {
    nodes: nodes,
    edges: edges,
  };
  var options = {
    width: '100%',
    height: s.windowHeight - iconHeight + "px",
    physics: {
      stabilization: false,
      barnesHut: {
        damping: 1
      }
    },
    nodes: {
      shape: 'dot',
      scaling: {
        min: 5,
        max: 50,
        label: {
          enabled: true,
          min: 6,
          max: 20,
          drawThreshold: 4
        },
        customScalingFunction: function(min, max, total, value) {
          if (max == min) {
            return 0.5;
          } else {
            var scale = 1 / (max - min);
            return Math.max(0, Math.sqrt((Math.abs(value) - min) * scale));
          }
        }
      },
    },
    edges: {
      arrows: {
        to: {
          enabled: true
        }
      }
    }
  };
  var network = new vis.Network(container, data, options);
};

function tileOptions(s) {
  var myMapOptions = {
    getTileUrl: function(coord, zoom) {
      return "{{.}}/maptile?pathselect=" + s.pathID + "&units=" + s.units + "&amt=" + s.amt + "&varname=" + s.resultVar + "&zoom=" + zoom + "&x=" + (coord.x) + "&y=" + (coord.y);
    },
    tileSize: new google.maps.Size(256, 256),
    isPng: true,
    opacity: 1,
    name: "custom"
  };
  var customMapType = new google.maps.ImageMapType(myMapOptions);
  return customMapType;
}

var newMap = function(s) {
  $('#viewer').css('height', s.windowHeight - iconHeight);

  var labelTiles = {
    getTileUrl: function(coord, zoom) {
      return "http://mt0.google.com/vt/v=apt.116&hl=en-US&" +
        "z=" + zoom + "&x=" + coord.x + "&y=" + coord.y + "&client=api";
    },
    tileSize: new google.maps.Size(256, 256),
    isPng: true
  };
  var googleLabelLayer = new google.maps.ImageMapType(labelTiles);

  var latlng = new google.maps.LatLng(40, -97);
  var mapOptions = {
    zoom: 5,
    center: latlng,
    mapTypeId: google.maps.MapTypeId.ROADMAP,
    panControl: true,
    streetViewControl: false
  };

  map = new google.maps.Map(document.getElementById('viewer'), mapOptions);

  //loadMapLegend(s);

  var customMapType = tileOptions(s);
  map.overlayMapTypes.insertAt(0, customMapType);
  map.overlayMapTypes.insertAt(1, googleLabelLayer);
};

var updateMap = function(s) {
  //loadMapLegend(s);
  var customMapType = tileOptions(s);
  map.overlayMapTypes.removeAt(0);
  map.overlayMapTypes.insertAt(0, customMapType);
};

var showOnlyOptionsSimilarToText = function(selectionEl, str, isCaseSensitive) {
  if (typeof isCaseSensitive == 'undefined')
    isCaseSensitive = true;
  if (isCaseSensitive)
    str = str.toLowerCase();

  var $el = $(selectionEl);

  $el.children("option:selected").removeAttr('selected');
  $el.val('');
  $el.children("option").hide();

  $el.children("option").filter(function() {
    var text = $(this).text();
    if (isCaseSensitive)
      text = text.toLowerCase();

    if (text.indexOf(str) > -1)
      return true;

    return false;
  }).show();

};
$(document).ready(function() {
  $('[data-toggle=offcanvas]').click(function() {
    $('.row-offcanvas').toggleClass('active');
  });

  var timeout;
  $("#SearchBox").on("keyup", function() {
    var userInput = $("#SearchBox").val();
    window.clearTimeout(timeout);
    timeout = window.setTimeout(function() {
      showOnlyOptionsSimilarToText($("#pathselect"), userInput, true);
    }, 500);

  });
});
