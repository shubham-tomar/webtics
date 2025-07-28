;(function (host) {
    const q = [];
    const flush = () =>
      navigator.sendBeacon &&
      navigator.sendBeacon(
        host + "/track",
        JSON.stringify(q.shift()) // one event per beacon for P1 simplicity
      );
  
    // Auto page‑view
    q.push({
      event: "page_view",
      ts: Date.now(),
      url: location.href,
      ref: document.referrer || "",
      props: {},
    });
    flush();
  
    // Example custom call available globally later
    window.ml = {
      track: (e, p = {}) => {
        q.push({ event: e, ts: Date.now(), url: location.href, ref: "", props: p });
        flush();
      },
    };
  })("http://localhost:8080");
  