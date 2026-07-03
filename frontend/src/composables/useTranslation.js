import { ref } from 'vue'

const translations = {
  en: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'Product Security Verification',
    locationPromptTitle: 'To protect you from counterfeit products',
    locationPromptIntro: ', we need location access to:',
    locationReason1: 'Detect suspicious distribution patterns',
    locationReason2: 'Track illegal product circulation',
    locationReason3: 'Verify authenticity at your location',
    locationDisclaimer: 'Your location data is only used for verification and will not be shared with third parties.',
    allowLocation: 'Allow Location Access',
    requestingPermission: 'Requesting permission...',

    // Geolocation overlay - Denied
    verificationFailed: 'Verification Failed',
    deniedMessage: 'Without location access, we cannot verify the authenticity of this product. This is a security measure to protect you from counterfeit products.',
    tryAgain: 'Try Again',
    deniedHint: "If the browser doesn't show a permission prompt, please check the location permission settings in your browser.",

    // Geolocation overlay - Blocked
    locationBlocked: 'Location Access Blocked',
    blockedMessage: 'You have blocked location access for this site. To verify this product, please enable location in your browser settings:',
    howToEnable: 'How to enable:',
    step1: "Click the lock/info icon in your browser's address bar",
    step2: 'Find "Location" in the permissions list',
    step3: 'Change it to "Allow"',
    step4: 'Refresh this page',
    refreshPage: 'Refresh Page',
  },

  // Indonesian (Indonesia)
  id: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'Verifikasi Keamanan Produk',
    locationPromptTitle: 'Untuk melindungi Anda dari produk palsu',
    locationPromptIntro: ', kami memerlukan akses lokasi untuk:',
    locationReason1: 'Mendeteksi pola distribusi mencurigakan',
    locationReason2: 'Melacak peredaran produk ilegal',
    locationReason3: 'Memverifikasi keaslian di lokasi Anda',
    locationDisclaimer: 'Data lokasi Anda hanya digunakan untuk verifikasi dan tidak akan dibagikan kepada pihak ketiga.',
    allowLocation: 'Izinkan Akses Lokasi',
    requestingPermission: 'Meminta izin...',

    // Geolocation overlay - Denied
    verificationFailed: 'Verifikasi Gagal',
    deniedMessage: 'Tanpa akses lokasi, kami tidak dapat memverifikasi keaslian produk ini. Ini adalah langkah keamanan untuk melindungi Anda dari produk palsu.',
    tryAgain: 'Coba Lagi',
    deniedHint: 'Jika browser tidak menampilkan permintaan izin, silakan periksa pengaturan izin lokasi di browser Anda.',

    // Geolocation overlay - Blocked
    locationBlocked: 'Akses Lokasi Diblokir',
    blockedMessage: 'Anda telah memblokir akses lokasi untuk situs ini. Untuk memverifikasi produk ini, silakan aktifkan lokasi di pengaturan browser:',
    howToEnable: 'Cara mengaktifkan:',
    step1: 'Klik ikon gembok/info di address bar browser',
    step2: 'Cari "Lokasi" di daftar izin',
    step3: 'Ubah menjadi "Izinkan"',
    step4: 'Refresh halaman ini',
    refreshPage: 'Refresh Halaman',
  },

  // Malay (Malaysia)
  ms: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'Pengesahan Keselamatan Produk',
    locationPromptTitle: 'Untuk melindungi anda daripada produk palsu',
    locationPromptIntro: ', kami memerlukan akses lokasi untuk:',
    locationReason1: 'Mengesan corak pengedaran yang mencurigakan',
    locationReason2: 'Menjejaki peredaran produk haram',
    locationReason3: 'Mengesahkan ketulenan di lokasi anda',
    locationDisclaimer: 'Data lokasi anda hanya digunakan untuk pengesahan dan tidak akan dikongsi dengan pihak ketiga.',
    allowLocation: 'Benarkan Akses Lokasi',
    requestingPermission: 'Meminta kebenaran...',

    // Geolocation overlay - Denied
    verificationFailed: 'Pengesahan Gagal',
    deniedMessage: 'Tanpa akses lokasi, kami tidak dapat mengesahkan ketulenan produk ini. Ini adalah langkah keselamatan untuk melindungi anda daripada produk palsu.',
    tryAgain: 'Cuba Lagi',
    deniedHint: 'Jika pelayar tidak menunjukkan permintaan kebenaran, sila semak tetapan kebenaran lokasi di pelayar anda.',

    // Geolocation overlay - Blocked
    locationBlocked: 'Akses Lokasi Disekat',
    blockedMessage: 'Anda telah menyekat akses lokasi untuk laman ini. Untuk mengesahkan produk ini, sila aktifkan lokasi di tetapan pelayar:',
    howToEnable: 'Cara mengaktifkan:',
    step1: 'Klik ikon kunci/info di bar alamat pelayar',
    step2: 'Cari "Lokasi" dalam senarai kebenaran',
    step3: 'Tukar kepada "Benarkan"',
    step4: 'Muat semula halaman ini',
    refreshPage: 'Muat Semula Halaman',
  },

  // Filipino/Tagalog (Philippines)
  tl: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'Pag-verify ng Seguridad ng Produkto',
    locationPromptTitle: 'Upang protektahan ka mula sa mga pekeng produkto',
    locationPromptIntro: ', kailangan namin ang access sa lokasyon upang:',
    locationReason1: 'Makita ang mga kahina-hinalang pattern ng distribusyon',
    locationReason2: 'Subaybayan ang iligal na sirkulasyon ng produkto',
    locationReason3: 'I-verify ang pagiging tunay sa iyong lokasyon',
    locationDisclaimer: 'Ang iyong data ng lokasyon ay ginagamit lamang para sa verification at hindi ibabahagi sa mga third party.',
    allowLocation: 'Payagan ang Access sa Lokasyon',
    requestingPermission: 'Humihiling ng pahintulot...',

    // Geolocation overlay - Denied
    verificationFailed: 'Nabigo ang Verification',
    deniedMessage: 'Kung walang access sa lokasyon, hindi namin ma-verify ang pagiging tunay ng produktong ito. Ito ay isang hakbang sa seguridad upang protektahan ka mula sa mga pekeng produkto.',
    tryAgain: 'Subukan Muli',
    deniedHint: 'Kung hindi nagpapakita ang browser ng prompt para sa pahintulot, pakisuri ang mga setting ng pahintulot sa lokasyon sa iyong browser.',

    // Geolocation overlay - Blocked
    locationBlocked: 'Na-block ang Access sa Lokasyon',
    blockedMessage: 'Na-block mo ang access sa lokasyon para sa site na ito. Upang ma-verify ang produktong ito, pakipagana ang lokasyon sa mga setting ng browser:',
    howToEnable: 'Paano paganahin:',
    step1: 'I-click ang lock/info icon sa address bar ng browser',
    step2: 'Hanapin ang "Location" sa listahan ng mga pahintulot',
    step3: 'Palitan ito sa "Allow"',
    step4: 'I-refresh ang pahinang ito',
    refreshPage: 'I-refresh ang Pahina',
  },

  // Thai (Thailand)
  th: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'การตรวจสอบความปลอดภัยของผลิตภัณฑ์',
    locationPromptTitle: 'เพื่อปกป้องคุณจากสินค้าปลอม',
    locationPromptIntro: ' เราต้องการการเข้าถึงตำแหน่งเพื่อ:',
    locationReason1: 'ตรวจจับรูปแบบการจัดจำหน่ายที่น่าสงสัย',
    locationReason2: 'ติดตามการหมุนเวียนสินค้าผิดกฎหมาย',
    locationReason3: 'ยืนยันความถูกต้องที่ตำแหน่งของคุณ',
    locationDisclaimer: 'ข้อมูลตำแหน่งของคุณใช้เพื่อการยืนยันเท่านั้นและจะไม่ถูกแบ่งปันกับบุคคลที่สาม',
    allowLocation: 'อนุญาตการเข้าถึงตำแหน่ง',
    requestingPermission: 'กำลังขออนุญาต...',

    // Geolocation overlay - Denied
    verificationFailed: 'การยืนยันล้มเหลว',
    deniedMessage: 'หากไม่มีการเข้าถึงตำแหน่ง เราไม่สามารถยืนยันความถูกต้องของผลิตภัณฑ์นี้ได้ นี่เป็นมาตรการรักษาความปลอดภัยเพื่อปกป้องคุณจากสินค้าปลอม',
    tryAgain: 'ลองอีกครั้ง',
    deniedHint: 'หากเบราว์เซอร์ไม่แสดงข้อความขออนุญาต โปรดตรวจสอบการตั้งค่าการอนุญาตตำแหน่งในเบราว์เซอร์ของคุณ',

    // Geolocation overlay - Blocked
    locationBlocked: 'การเข้าถึงตำแหน่งถูกบล็อก',
    blockedMessage: 'คุณได้บล็อกการเข้าถึงตำแหน่งสำหรับเว็บไซต์นี้ เพื่อยืนยันผลิตภัณฑ์นี้ โปรดเปิดใช้งานตำแหน่งในการตั้งค่าเบราว์เซอร์:',
    howToEnable: 'วิธีเปิดใช้งาน:',
    step1: 'คลิกไอคอนล็อค/ข้อมูลในแถบที่อยู่ของเบราว์เซอร์',
    step2: 'ค้นหา "ตำแหน่ง" ในรายการสิทธิ์',
    step3: 'เปลี่ยนเป็น "อนุญาต"',
    step4: 'รีเฟรชหน้านี้',
    refreshPage: 'รีเฟรชหน้า',
  },

  // Vietnamese (Vietnam)
  vi: {
    // Geolocation overlay - Pending/Requesting
    securityVerification: 'Xác minh bảo mật sản phẩm',
    locationPromptTitle: 'Để bảo vệ bạn khỏi sản phẩm giả',
    locationPromptIntro: ', chúng tôi cần quyền truy cập vị trí để:',
    locationReason1: 'Phát hiện các mô hình phân phối đáng ngờ',
    locationReason2: 'Theo dõi lưu hành sản phẩm bất hợp pháp',
    locationReason3: 'Xác minh tính xác thực tại vị trí của bạn',
    locationDisclaimer: 'Dữ liệu vị trí của bạn chỉ được sử dụng để xác minh và sẽ không được chia sẻ với bên thứ ba.',
    allowLocation: 'Cho phép truy cập vị trí',
    requestingPermission: 'Đang yêu cầu quyền...',

    // Geolocation overlay - Denied
    verificationFailed: 'Xác minh thất bại',
    deniedMessage: 'Không có quyền truy cập vị trí, chúng tôi không thể xác minh tính xác thực của sản phẩm này. Đây là biện pháp bảo mật để bảo vệ bạn khỏi sản phẩm giả.',
    tryAgain: 'Thử lại',
    deniedHint: 'Nếu trình duyệt không hiển thị lời nhắc xin quyền, vui lòng kiểm tra cài đặt quyền vị trí trong trình duyệt của bạn.',

    // Geolocation overlay - Blocked
    locationBlocked: 'Quyền truy cập vị trí bị chặn',
    blockedMessage: 'Bạn đã chặn quyền truy cập vị trí cho trang web này. Để xác minh sản phẩm này, vui lòng bật vị trí trong cài đặt trình duyệt:',
    howToEnable: 'Cách bật:',
    step1: 'Nhấp vào biểu tượng khóa/thông tin trên thanh địa chỉ của trình duyệt',
    step2: 'Tìm "Vị trí" trong danh sách quyền',
    step3: 'Đổi thành "Cho phép"',
    step4: 'Làm mới trang này',
    refreshPage: 'Làm mới trang',
  }
}

/**
 * Composable for browser language detection and translation
 * Supports: English (default), Indonesian, Malay, Filipino, Thai, Vietnamese
 */
export function useTranslation() {
  // Detect browser language
  const browserLang = navigator.language || navigator.userLanguage || 'en'

  // Determine language code
  let detectedLang = 'en' // default
  if (browserLang.startsWith('id')) {
    detectedLang = 'id'
  } else if (browserLang.startsWith('ms')) {
    detectedLang = 'ms'
  } else if (browserLang.startsWith('tl') || browserLang.startsWith('fil')) {
    detectedLang = 'tl'
  } else if (browserLang.startsWith('th')) {
    detectedLang = 'th'
  } else if (browserLang.startsWith('vi')) {
    detectedLang = 'vi'
  }

  const lang = ref(detectedLang)

  /**
   * Get translation for a key
   * Falls back to English if translation not found
   * @param {string} key - Translation key
   * @returns {string} Translated text
   */
  const t = (key) => {
    return translations[lang.value]?.[key] || translations['en'][key] || key
  }

  return { t, lang }
}
