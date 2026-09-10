import json
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".")
en_path = root / "lib/l10n/arb/app_en.arb"
vi_path = root / "lib/l10n/arb/app_vi.arb"

translations = {
    "dialogImportPlaylistTitle": "Nhập danh sách phát",
    "dialogImportPlaylistMessage": "Tìm thấy {count} bài hát trong tệp danh sách phát. Thêm chúng vào hàng chờ tải xuống?",
    "csvImportTracks": "{count} bài hát từ CSV",
    "collectionExportM3u": "Xuất dạng M3U8",
    "collectionExportM3uDone": "Đã xuất {exported}/{total} bài hát",
    "collectionExportM3uNone": "Không có tệp nhạc đã tải để xuất",
    "collectionExportM3uFailed": "Xuất danh sách phát thất bại",
    "trackOpenOn": "Mở bằng...",
    "trackOpenOnNoLinks": "Không tìm thấy liên kết nền tảng cho bài hát này.",
    "libraryReviewDuplicates": "Kiểm tra bản trùng",
    "libraryReviewDuplicatesSubtitle": "Tìm các bài hát được lưu nhiều lần",
    "duplicatesTitle": "Bản trùng",
    "duplicatesEmpty": "Không tìm thấy bài hát bị trùng.",
    "duplicatesKeepBest": "Giữ bản tốt nhất",
    "duplicatesKeepBestAll": "Giữ bản tốt nhất cho tất cả",
    "duplicatesKeepBestAllMessage": "Giữ bản tốt nhất của {groupCount} bài hát và xóa {count} bản chất lượng thấp hơn?\n\nCác tệp bị xóa sẽ được xóa khỏi bộ nhớ.",
    "duplicatesKeepBestMessage": "Xóa {count} bản chất lượng thấp hơn của \"{trackName}\"?",
    "duplicatesDeleteCopyMessage": "Xóa bản sao này của \"{trackName}\"?",
    "snackbarAddedToQueue": "Đã thêm \"{trackName}\" vào hàng chờ",
    "snackbarAddedTracksToQueue": "Đã thêm {count} bài hát vào hàng chờ",
    "snackbarAlreadyDownloaded": "\"{trackName}\" đã được tải xuống",
    "snackbarAlreadyInLibrary": "\"{trackName}\" đã có trong thư viện của bạn",
    "snackbarHistoryCleared": "Đã xóa lịch sử",
    "snackbarDeletedTracks": "Đã xóa {count} bài hát",
    "snackbarCannotOpenFile": "Không thể mở tệp: {error}",
    "snackbarViewQueue": "Xem hàng chờ",
    "snackbarUrlCopied": "Đã sao chép liên kết {platform} vào bộ nhớ tạm",
    "snackbarFileNotFound": "Không tìm thấy tệp",
    "snackbarSelectExtFile": "Vui lòng chọn tệp .spotiflac-ext",
    "snackbarProviderPrioritySaved": "Đã lưu thứ tự ưu tiên nguồn",
    "snackbarMetadataProviderSaved": "Đã lưu thứ tự ưu tiên nguồn thông tin bài hát",
    "snackbarExtensionInstalled": "Đã cài {extensionName}.",
    "snackbarExtensionUpdated": "Đã cập nhật {extensionName}.",
    "snackbarFailedToInstall": "Không thể cài tiện ích",
    "snackbarFailedToUpdate": "Không thể cập nhật tiện ích",
    "errorRateLimited": "Đã đạt giới hạn yêu cầu",
    "errorRateLimitedMessage": "Có quá nhiều yêu cầu. Vui lòng chờ một lúc rồi tìm kiếm lại.",
    "errorNoTracksFound": "Không tìm thấy bài hát",
    "searchEmptyResultSubtitle": "Hãy thử từ khóa khác",
    "errorUrlNotRecognized": "Không nhận diện được liên kết",
    "errorUrlNotRecognizedMessage": "Liên kết này chưa được hỗ trợ. Hãy kiểm tra URL và đảm bảo bạn đã cài tiện ích tương thích.",
    "errorUrlFetchFailed": "Không thể tải nội dung từ liên kết này. Vui lòng thử lại.",
    "errorMissingExtensionSource": "Không thể tải {item}: thiếu nguồn tiện ích",
    "actionPause": "Tạm dừng",
    "actionResume": "Tiếp tục",
    "actionCancel": "Hủy",
    "actionSelectAll": "Chọn tất cả",
    "actionDeselect": "Bỏ chọn",
    "selectionSelected": "Đã chọn {count}",
    "selectionAllSelected": "Đã chọn tất cả bài hát",
    "selectionSelectToDelete": "Chọn bài hát cần xóa",
    "progressFetchingMetadata": "Đang lấy thông tin bài hát... {current}/{total}",
}

with en_path.open(encoding="utf-8") as handle:
    en = json.load(handle)
with vi_path.open(encoding="utf-8") as handle:
    vi = json.load(handle)

missing = [key for key in translations if key not in en]
if missing:
    raise SystemExit(f"English ARB keys missing: {missing}")

for key, value in translations.items():
    vi[key] = value
    metadata_key = f"@{key}"
    if metadata_key in en:
        vi[metadata_key] = en[metadata_key]

vi["@@last_modified"] = "2026-09-10"
vi_path.write_text(json.dumps(vi, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

message_keys = [key for key in en if not key.startswith("@")]
present = sum(1 for key in message_keys if key in vi)
print(f"Added/updated {len(translations)} Vietnamese messages")
print(f"Vietnamese ARB now contains {present}/{len(message_keys)} source message keys")
