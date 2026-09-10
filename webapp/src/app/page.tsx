const quickItems = [
  { title: "Bài hát", caption: "Tìm và phát trực tiếp" },
  { title: "Album", caption: "Xem toàn bộ album" },
  { title: "Danh sách phát", caption: "Mở playlist từ liên kết" },
];

export default function HomePage() {
  return (
    <main className="app-shell">
      <aside className="sidebar" aria-label="Điều hướng chính">
        <div className="brand"><span className="brand-mark">S</span><span>SpotiFLAC</span></div>
        <nav>
          <a className="nav-item active" href="#home">Trang chủ</a>
          <a className="nav-item" href="#library">Thư viện</a>
          <a className="nav-item" href="#extensions">Kho tiện ích</a>
          <a className="nav-item" href="#settings">Cài đặt</a>
        </nav>
      </aside>

      <section className="content" id="home">
        <header className="topbar">
          <div>
            <p className="eyebrow">SPOTIFLAC WEB</p>
            <h1>Nghe nhạc theo cách của bạn</h1>
          </div>
          <button className="install-button" type="button" aria-label="Cài SpotiFLAC PWA">Cài ứng dụng</button>
        </header>

        <form className="search-card" action="#" onSubmit={undefined}>
          <label htmlFor="search">Tìm nhạc hoặc dán liên kết</label>
          <div className="search-row">
            <input id="search" name="q" placeholder="Tên bài hát, album, nghệ sĩ hoặc URL…" autoComplete="off" />
            <button type="submit">Tìm kiếm</button>
          </div>
          <p>Hỗ trợ bài hát, album, danh sách phát và nghệ sĩ. Kết nối provider sẽ được thêm ở lớp API riêng.</p>
        </form>

        <section aria-labelledby="quick-title">
          <div className="section-title"><h2 id="quick-title">Khám phá</h2><span>Responsive cho PC · Tablet · Mobile</span></div>
          <div className="card-grid">
            {quickItems.map((item) => (
              <article className="media-card" key={item.title}>
                <div className="media-art" aria-hidden="true">♪</div>
                <h3>{item.title}</h3>
                <p>{item.caption}</p>
              </article>
            ))}
          </div>
        </section>
      </section>

      <footer className="player" aria-label="Trình phát nhạc">
        <div className="track-placeholder"><div className="mini-art">♪</div><div><strong>Chưa phát bài hát</strong><span>Chọn một bài để bắt đầu</span></div></div>
        <div className="player-controls"><button aria-label="Bài trước">‹</button><button className="play" aria-label="Phát">▶</button><button aria-label="Bài tiếp">›</button></div>
        <div className="player-status">00:00 / 00:00</div>
      </footer>
    </main>
  );
}
