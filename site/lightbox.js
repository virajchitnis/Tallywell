(function () {
  const overlay = document.createElement('div');
  overlay.id = 'img-overlay';
  overlay.setAttribute('role', 'dialog');
  overlay.setAttribute('aria-modal', 'true');
  overlay.setAttribute('aria-label', 'Image preview');

  const img = document.createElement('img');
  img.alt = 'Full-size screenshot';

  const btn = document.createElement('button');
  btn.className = 'overlay-close';
  btn.setAttribute('aria-label', 'Close');
  btn.textContent = '×';

  overlay.appendChild(img);
  overlay.appendChild(btn);
  document.body.appendChild(overlay);

  function open(src) {
    img.src = src;
    overlay.classList.add('open');
    document.body.style.overflow = 'hidden';
  }

  function close() {
    overlay.classList.remove('open');
    document.body.style.overflow = '';
  }

  document.addEventListener('click', function (e) {
    if (e.target.closest('.screenshot img, .preview-img img')) {
      open(e.target.closest('img').src);
      return;
    }
    if (e.target === overlay || e.target === btn) close();
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') close();
  });
})();
