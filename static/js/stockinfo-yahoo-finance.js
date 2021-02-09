function plotIndividualGraph(stocklist, companyinfo, ratiolist) {
  var datelist = [];
  var closelist = [];
  var openlist = [];
  var highlist = [];
  var lowlist = [];
  var volumelist = [];
  for (var i = 0; i < stocklist.length; i++) {
    datelist.push(stocklist[i].date);
    closelist.push(stocklist[i].close);
    openlist.push(stocklist[i].open);
    highlist.push(stocklist[i].high);
    lowlist.push(stocklist[i].low);
    volumelist.push(stocklist[i].volume);
  }
  console.log(stocklist);
  console.log(datelist);
  var trace1 = {
    x: datelist,
    close: closelist,
    decreasing: { line: { color: "red" } },
    high: highlist,
    increasing: { line: { color: "#17BECF" } },
    line: { color: "rgba(31,119,180,1)" },
    low: lowlist,
    open: openlist,
    type: "candlestick",
    xaxis: "x",
    yaxis: "y",
  };

  var trace2 = {
    x: datelist,
    y: volumelist,
    name: "yaxis2 data",
    yaxis: "y2",
    type: "bar",
    opacity: 0.4,
  };

  var titlelist = [
    "<b>COMPANY</b>",
    "<b>SYMBOL</b>",
    "<b>CODE</b>",
    "<b>MARKET</b>",
    "<b>SECTOR</b>",
    "<b>SUB SECTOR</b>",
    "<b>MARKET CAP</b>",
  ];
  var contentlist = [
    companyinfo.Company,
    companyinfo.Symbol,
    companyinfo.StockCode,
    companyinfo.Market,
    companyinfo.Sector,
    companyinfo.SubSector,
    companyinfo.MarketCap,
  ];

  for (var i = 0; i < ratiolist.length; i++) {
    console.log(ratiolist[i].Title);
    titlelist.push("<b>" + ratiolist[i].Title + "</b>");
    contentlist.push(ratiolist[i].Content);
  }

  var values = [titlelist, contentlist];

  var table = {
    type: "table",
    header: {
      values: [["<b>TITLE</b>"], ["<b>DATA</b>"]],
      align: "center",
      line: { width: 1, color: "black" },
      fill: { color: "grey" },
      font: { family: "Arial", size: 12, color: "white" },
    },
    cells: {
      values: values,
      align: "center",
      line: { color: "black", width: 1 },
      font: { family: "Arial", size: 11, color: ["black"] },
    },
    xaxis: "x3",
    yaxis: "y3",
    domain: { x: [0.7, 1], y: [0, 1] },
  };

  var layout = {
    // grid: {rows: 1, columns: 2, pattern: 'independent'},
    // dragmode: 'zoom',
    margin: {
      // r: 10,
      t: 25,
      b: 40,
      // l: 60
    },
    showlegend: false,
    xaxis: {
      // autorange: true,
      domain: [0, 0.65],
      range: [datelist[datelist.length - 350], datelist[datelist.length - 1]],
      // rangeslider: {range: ['2017-01-03 12:00', '2017-02-15 12:00']},
      title: "Date",
      type: "date",
    },
    yaxis: {
      autorange: true,
      domain: [0, 1],
      // range: [114.609999778, 137.410004222],
      type: "linear",
    },
    yaxis2: {
      title: "Volume",
      autorange: true,
      domain: [0, 1],
      titlefont: { color: "#ff7f0e" },
      tickfont: { color: "#ff7f0e" },
      // anchor: 'free',
      overlaying: "y",
      side: "right",
      position: 0.63,
    },
  };

  var data = [trace1, trace2, table];
  Plotly.newPlot("stock-chart", data, layout);
}
