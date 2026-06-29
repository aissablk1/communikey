/* csend — landing. Vanilla, zéro dépendance, zéro appel réseau.
   Deux effets nommés seulement :
   (1) spotlight radial réactif au pointeur (transform → GPU)
   (2) une séquence de reveal au scroll (opacity + translateY)
   prefers-reduced-motion : tout est neutralisé. */
(function () {
  "use strict";

  var reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---- 1. Reveal au scroll (IntersectionObserver) ---- */
  var revealables = Array.prototype.slice.call(document.querySelectorAll(".reveal"));

  if (reduce || !("IntersectionObserver" in window)) {
    // Pas de motion : tout est visible immédiatement.
    revealables.forEach(function (el) { el.classList.add("in-view"); });
  } else {
    var io = new IntersectionObserver(function (entries, obs) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        var el = entry.target;
        var d = parseInt(el.getAttribute("data-d") || "0", 10);
        el.style.transitionDelay = (d * 90) + "ms";
        el.classList.add("in-view");
        obs.unobserve(el);
      });
    }, { rootMargin: "0px 0px -8% 0px", threshold: 0.12 });
    revealables.forEach(function (el) { io.observe(el); });
  }

  /* ---- 2. Spotlight réactif au pointeur ---- */
  var spot = document.querySelector(".spotlight");
  if (spot && !reduce && window.matchMedia("(pointer:fine)").matches) {
    var tx = 0, ty = 0, cx = 0, cy = 0, queued = false;
    function apply() {
      queued = false;
      // suivi amorti vers la cible
      cx += (tx - cx) * 0.16;
      cy += (ty - cy) * 0.16;
      spot.style.transform = "translate3d(" + cx.toFixed(1) + "px," + cy.toFixed(1) + "px,0)";
      if (Math.abs(tx - cx) > 0.4 || Math.abs(ty - cy) > 0.4) {
        requestAnimationFrame(apply);
      }
    }
    window.addEventListener("pointermove", function (e) {
      // décalage limité depuis le centre — un glow qui « respire », pas qui saute
      tx = (e.clientX - window.innerWidth / 2) * 0.06;
      ty = (e.clientY - window.innerHeight / 2) * 0.06;
      if (!queued) { queued = true; requestAnimationFrame(apply); }
    }, { passive: true });
  }

  /* ---- 3. Copier la commande d’installation ---- */
  var copyBtn = document.querySelector(".cmd__copy");
  var cmdBox = document.querySelector(".cmd");
  if (copyBtn && cmdBox) {
    var original = copyBtn.textContent;
    copyBtn.addEventListener("click", function () {
      var text = cmdBox.getAttribute("data-copy") || "";
      var done = function () {
        copyBtn.textContent = "Copié ✓";
        copyBtn.classList.add("is-copied");
        setTimeout(function () {
          copyBtn.textContent = original;
          copyBtn.classList.remove("is-copied");
        }, 1600);
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done).catch(fallback);
      } else { fallback(); }
      function fallback() {
        var ta = document.createElement("textarea");
        ta.value = text; ta.setAttribute("readonly", "");
        ta.style.position = "absolute"; ta.style.left = "-9999px";
        document.body.appendChild(ta); ta.select();
        try { document.execCommand("copy"); done(); } catch (e) { /* silencieux */ }
        document.body.removeChild(ta);
      }
    });
  }
})();
