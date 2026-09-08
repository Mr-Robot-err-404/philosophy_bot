(function () {
  var cells = Array.prototype.slice.call(
    document.querySelectorAll(".countdown[data-remaining]")
  );
  if (cells.length === 0) return;

  var state = cells.map(function (cell) {
    return { cell: cell, left: parseInt(cell.dataset.remaining, 10) || 0 };
  });

  function format(total) {
    if (total <= 0) return "due";
    var h = Math.floor(total / 3600);
    var m = Math.floor((total % 3600) / 60);
    var s = total % 60;
    var pad = function (n) {
      return n < 10 ? "0" + n : String(n);
    };
    if (h > 0) return h + ":" + pad(m) + ":" + pad(s);
    return m + ":" + pad(s);
  }

  function render() {
    state.forEach(function (job) {
      job.cell.textContent = format(job.left);
      job.cell.classList.toggle("due", job.left <= 0);
    });
  }

  render();

  setInterval(function () {
    state.forEach(function (job) {
      if (job.left > 0) job.left -= 1;
    });
    render();
  }, 1000);
})();
