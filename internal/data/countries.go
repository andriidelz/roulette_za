package data

import (
	"strings"
	"time"
)

// Country представляет информацию о стране
type Country struct {
	Code     string // ISO 3166-1 alpha-2
	Emoji    string
	Order    int
	Name     string
	Favorite bool
	Region   string
	Timezone string
}

// Countries содержит список стран
var Countries = []Country{

	// Favorite

	{"UA", "🇺🇦", 1, "Ukraine", true, "Europe", "Europe/Kiev"},
	{"PL", "🇵🇱", 2, "Poland", true, "Europe", "Europe/Warsaw"},
	{"DE", "🇩🇪", 3, "Germany", true, "Europe", "Europe/Berlin"},
	{"FR", "🇫🇷", 4, "France", true, "Europe", "Europe/Paris"},
	{"IT", "🇮🇹", 5, "Italy", true, "Europe", "Europe/Rome"},
	{"GB", "🇬🇧", 6, "United Kingdom of Great Britain and Northern Ireland", true, "Europe", "Europe/London"},
	{"AT", "🇦🇹", 7, "Austria", false, "Europe", "Europe/Vienna"},
	{"CY", "🇨🇾", 8, "Cyprus", true, "Europe", "Asia/Nicosia"},
	{"GR", "🇬🇷", 9, "Greece", true, "Europe", "Europe/Athens"},
	{"HR", "🇭🇷", 10, "Croatia", true, "Europe", "Europe/Zagreb"},
	{"CZ", "🇨🇿", 11, "Czechia", true, "Europe", "Europe/Prague"},
	{"SK", "🇸🇰", 12, "Slovakia", true, "Europe", "Europe/Bratislava"},
	{"RO", "🇷🇴", 13, "Romania", true, "Europe", "Europe/Bucharest"},
	{"HU", "🇭🇺", 14, "Hungary", true, "Europe", "Europe/Budapest"},
	{"RS", "🇷🇸", 15, "Serbia", true, "Europe", "Europe/Belgrade"},
	{"MD", "🇲🇩", 16, "Moldova, Republic of Moldova", true, "Europe", "Europe/Chisinau"},
	{"ES", "🇪🇸", 17, "Spain", true, "Europe", "Europe/Madrid"},
	{"PT", "🇵🇹", 18, "Portugal", true, "Europe", "Europe/Lisbon"},
	{"EE", "🇪🇪", 19, "Estonia", true, "Europe", "Europe/Tallinn"},
	{"LV", "🇱🇻", 20, "Latvia", true, "Europe", "Europe/Riga"},
	{"LT", "🇱🇹", 21, "Lithuania", true, "Europe", "Europe/Vilnius"},
	{"GE", "🇬🇪", 22, "Georgia", true, "Asia", "Asia/Tbilisi"},
	{"KZ", "🇰🇿", 23, "Kazakhstan", true, "Asia", "Asia/Almaty"},
	{"UZ", "🇺🇿", 24, "Uzbekistan", true, "Asia", "Asia/Tashkent"},
	{"BY", "❌", 25, "Belarus", false, "Europe", "Europe/Minsk"},
	{"RU", "❌", 26, "Russian Federation", false, "Europe", "Europe/Moscow"},

	// Non favorite

	{"AE", "🇦🇪", 0, "United Arab Emirates", false, "Asia", "Asia/Dubai"},
	{"AU", "🇦🇺", 0, "Australia", false, "Oceania", "Australia/Sydney"},
	{"BR", "🇧🇷", 0, "Brazil", false, "Americas", "America/Sao_Paulo"},
	{"DK", "🇩🇰", 0, "Denmark", false, "Europe", "Europe/Copenhagen"},
	{"EG", "🇪🇬", 0, "Egypt", false, "Africa", "Africa/Cairo"},
	{"FI", "🇫🇮", 0, "Finland", false, "Europe", "Europe/Helsinki"},
	{"IL", "🇮🇱", 0, "Israel", false, "Asia", "Asia/Jerusalem"},
	{"IN", "🇮🇳", 0, "India", false, "Asia", "Asia/Kolkata"},
	{"NL", "🇳🇱", 0, "Netherlands, Kingdom of the Netherlands", false, "Europe", "Europe/Amsterdam"},
	{"NO", "🇳🇴", 0, "Norway", false, "Europe", "Europe/Oslo"},
	{"SA", "🇸🇦", 0, "Saudi Arabia", false, "Asia", "Asia/Riyadh"},
	{"SE", "🇸🇪", 0, "Sweden", false, "Europe", "Europe/Stockholm"},
	{"TR", "🇹🇷", 0, "Turkey, Republic of Türkiye", false, "Asia", "Europe/Istanbul"},
	{"US", "🇺🇸", 0, "United States of America", false, "Americas", "America/New_York"},

	{"AD", "🇦🇩", 0, "Andorra", false, "Europe", "Europe/Andorra"},
	{"AF", "🇦🇫", 0, "Afghanistan", false, "Asia", "Asia/Kabul"},
	{"AG", "🇦🇬", 0, "Antigua and Barbuda", false, "Americas", "America/Antigua"},
	{"AI", "🇦🇮", 0, "Anguilla", false, "Americas", "America/Anguilla"},
	{"AL", "🇦🇱", 0, "Albania", false, "Europe", "Europe/Tirane"},
	{"AM", "🇦🇲", 0, "Armenia", false, "Asia", "Asia/Yerevan"},
	{"AO", "🇦🇴", 0, "Angola", false, "Africa", "Africa/Luanda"},
	{"AQ", "🇦🇶", 0, "Antarctica", false, "Antarctica", "Antarctica/McMurdo"},
	{"AR", "🇦🇷", 0, "Argentina", false, "Americas", "America/Argentina/Buenos_Aires"},
	{"AS", "🇦🇸", 0, "American Samoa", false, "Oceania", "Pacific/Pago_Pago"},
	{"AW", "🇦🇼", 0, "Aruba", false, "Americas", "America/Aruba"},
	{"AX", "🇦🇽", 0, "Åland Islands", false, "Europe", "Europe/Mariehamn"},
	{"AZ", "🇦🇿", 0, "Azerbaijan", false, "Asia", "Asia/Baku"},
	{"BA", "🇧🇦", 0, "Bosnia and Herzegovina", false, "Europe", "Europe/Sarajevo"},
	{"BB", "🇧🇧", 0, "Barbados", false, "Americas", "America/Barbados"},
	{"BD", "🇧🇩", 0, "Bangladesh", false, "Asia", "Asia/Dhaka"},
	{"BE", "🇧🇪", 0, "Belgium", false, "Europe", "Europe/Brussels"},
	{"BF", "🇧🇫", 0, "Burkina Faso", false, "Africa", "Africa/Ouagadougou"},
	{"BG", "🇧🇬", 0, "Bulgaria", false, "Europe", "Europe/Sofia"},
	{"BH", "🇧🇭", 0, "Bahrain", false, "Asia", "Asia/Bahrain"},
	{"BI", "🇧🇮", 0, "Burundi", false, "Africa", "Africa/Bujumbura"},
	{"BJ", "🇧🇯", 0, "Benin", false, "Africa", "Africa/Porto-Novo"},
	{"BL", "🇧🇱", 0, "Saint Barthélemy", false, "Americas", "America/St_Barthelemy"},
	{"BM", "🇧🇲", 0, "Bermuda", false, "Americas", "Atlantic/Bermuda"},
	{"BN", "🇧🇳", 0, "Brunei Darussalam", false, "Asia", "Asia/Brunei"},
	{"BO", "🇧🇴", 0, "Bolivia, Plurinational State of", false, "Americas", "America/La_Paz"},
	{"BQ", "🇧🇶", 0, "Bonaire, Sint Eustatius and Saba", false, "Americas", "America/Kralendijk"},
	{"BS", "🇧🇸", 0, "Bahamas", false, "Americas", "America/Nassau"},
	{"BT", "🇧🇹", 0, "Bhutan", false, "Asia", "Asia/Thimphu"},
	{"BV", "🇧🇻", 0, "Bouvet Island", false, "Antarctica", "Antarctica/Syowa"},
	{"BW", "🇧🇼", 0, "Botswana", false, "Africa", "Africa/Gaborone"},
	{"BZ", "🇧🇿", 0, "Belize", false, "Americas", "America/Belize"},
	{"CA", "🇨🇦", 0, "Canada", false, "Americas", "America/Toronto"},
	{"CC", "🇨🇨", 0, "Cocos (Keeling) Islands", false, "Asia", "Indian/Cocos"},
	{"CD", "🇨🇩", 0, "Congo, Democratic Republic of the", false, "Africa", "Africa/Kinshasa"},
	{"CF", "🇨🇫", 0, "Central African Republic", false, "Africa", "Africa/Bangui"},
	{"CG", "🇨🇬", 0, "Congo", false, "Africa", "Africa/Brazzaville"},
	{"CH", "🇨🇭", 0, "Switzerland", false, "Europe", "Europe/Zurich"},
	{"CI", "🇨🇮", 0, "Côte d'Ivoire", false, "Africa", "Africa/Abidjan"},
	{"CK", "🇨🇰", 0, "Cook Islands", false, "Oceania", "Pacific/Rarotonga"},
	{"CL", "🇨🇱", 0, "Chile", false, "Americas", "America/Santiago"},
	{"CM", "🇨🇲", 0, "Cameroon", false, "Africa", "Africa/Douala"},
	{"CN", "🇨🇳", 0, "China", false, "Asia", "Asia/Shanghai"},
	{"CO", "🇨🇴", 0, "Colombia", false, "Americas", "America/Bogota"},
	{"CR", "🇨🇷", 0, "Costa Rica", false, "Americas", "America/Costa_Rica"},
	{"CU", "🇨🇺", 0, "Cuba", false, "Americas", "America/Havana"},
	{"CV", "🇨🇻", 0, "Cabo Verde", false, "Africa", "Atlantic/Cape_Verde"},
	{"CW", "🇨🇼", 0, "Curaçao", false, "Americas", "America/Curacao"},
	{"CX", "🇨🇽", 0, "Christmas Island", false, "Asia", "Indian/Christmas"},
	{"DJ", "🇩🇯", 0, "Djibouti", false, "Africa", "Africa/Djibouti"},
	{"DM", "🇩🇲", 0, "Dominica", false, "Americas", "America/Dominica"},
	{"DO", "🇩🇴", 0, "Dominican Republic", false, "Americas", "America/Santo_Domingo"},
	{"DZ", "🇩🇿", 0, "Algeria", false, "Africa", "Africa/Algiers"},
	{"EC", "🇪🇨", 0, "Ecuador", false, "Americas", "America/Guayaquil"},
	{"EH", "🇪🇭", 0, "Western Sahara", false, "Africa", "Africa/El_Aaiun"},
	{"ER", "🇪🇷", 0, "Eritrea", false, "Africa", "Africa/Asmara"},
	{"ET", "🇪🇹", 0, "Ethiopia", false, "Africa", "Africa/Addis_Ababa"},
	{"FJ", "🇫🇯", 0, "Fiji", false, "Oceania", "Pacific/Fiji"},
	{"FK", "🇫🇰", 0, "Falkland Islands (Malvinas)", false, "Americas", "Atlantic/Stanley"},
	{"FM", "🇫🇲", 0, "Micronesia, Federated States of", false, "Oceania", "Pacific/Chuuk"},
	{"FO", "🇫🇴", 0, "Faroe Islands", false, "Europe", "Atlantic/Faroe"},
	{"GA", "🇬🇦", 0, "Gabon", false, "Africa", "Africa/Libreville"},
	{"GD", "🇬🇩", 0, "Grenada", false, "Americas", "America/Grenada"},
	{"GF", "🇬🇫", 0, "French Guiana", false, "Americas", "America/Cayenne"},
	{"GG", "🇬🇬", 0, "Guernsey", false, "Europe", "Europe/Guernsey"},
	{"GH", "🇬🇭", 0, "Ghana", false, "Africa", "Africa/Accra"},
	{"GI", "🇬🇮", 0, "Gibraltar", false, "Europe", "Europe/Gibraltar"},
	{"GL", "🇬🇱", 0, "Greenland", false, "Americas", "America/Nuuk"},
	{"GM", "🇬🇲", 0, "Gambia", false, "Africa", "Africa/Banjul"},
	{"GN", "🇬🇳", 0, "Guinea", false, "Africa", "Africa/Conakry"},
	{"GP", "🇬🇵", 0, "Guadeloupe", false, "Americas", "America/Guadeloupe"},
	{"GQ", "🇬🇶", 0, "Equatorial Guinea", false, "Africa", "Africa/Malabo"},
	{"GS", "🇬🇸", 0, "South Georgia and the South Sandwich Islands", false, "Antarctica", "Atlantic/South_Georgia"},
	{"GT", "🇬🇹", 0, "Guatemala", false, "Americas", "America/Guatemala"},
	{"GU", "🇬🇺", 0, "Guam", false, "Oceania", "Pacific/Guam"},
	{"GW", "🇬🇼", 0, "Guinea-Bissau", false, "Africa", "Africa/Bissau"},
	{"GY", "🇬🇾", 0, "Guyana", false, "Americas", "America/Guyana"},
	{"HK", "🇭🇰", 0, "Hong Kong", false, "Asia", "Asia/Hong_Kong"},
	{"HM", "🇭🇲", 0, "Heard Island and McDonald Islands", false, "Antarctica", "Indian/Kerguelen"},
	{"HN", "🇭🇳", 0, "Honduras", false, "Americas", "America/Tegucigalpa"},
	{"HT", "🇭🇹", 0, "Haiti", false, "Americas", "America/Port-au-Prince"},
	{"ID", "🇮🇩", 0, "Indonesia", false, "Asia", "Asia/Jakarta"},
	{"IE", "🇮🇪", 0, "Ireland", false, "Europe", "Europe/Dublin"},
	{"IM", "🇮🇲", 0, "Isle of Man", false, "Europe", "Europe/Isle_of_Man"},
	{"IO", "🇮🇴", 0, "British Indian Ocean Territory", false, "Asia", "Indian/Chagos"},
	{"IQ", "🇮🇶", 0, "Iraq", false, "Asia", "Asia/Baghdad"},
	{"IR", "🇮🇷", 0, "Iran, Islamic Republic of", false, "Asia", "Asia/Tehran"},
	{"IS", "🇮🇸", 0, "Iceland", false, "Europe", "Atlantic/Reykjavik"},
	{"JE", "🇯🇪", 0, "Jersey", false, "Europe", "Europe/Jersey"},
	{"JM", "🇯🇲", 0, "Jamaica", false, "Americas", "America/Jamaica"},
	{"JO", "🇯🇴", 0, "Jordan", false, "Asia", "Asia/Amman"},
	{"JP", "🇯🇵", 0, "Japan", false, "Asia", "Asia/Tokyo"},
	{"KE", "🇰🇪", 0, "Kenya", false, "Africa", "Africa/Nairobi"},
	{"KG", "🇰🇬", 0, "Kyrgyzstan", false, "Asia", "Asia/Bishkek"},
	{"KH", "🇰🇭", 0, "Cambodia", false, "Asia", "Asia/Phnom_Penh"},
	{"KI", "🇰🇮", 0, "Kiribati", false, "Oceania", "Pacific/Tarawa"},
	{"KM", "🇰🇲", 0, "Comoros", false, "Africa", "Indian/Comoro"},
	{"KN", "🇰🇳", 0, "Saint Kitts and Nevis", false, "Americas", "America/St_Kitts"},
	{"KP", "🇰🇵", 0, "Korea, Democratic People's Republic of", false, "Asia", "Asia/Pyongyang"},
	{"KR", "🇰🇷", 0, "Korea, Republic of", false, "Asia", "Asia/Seoul"},
	{"KW", "🇰🇼", 0, "Kuwait", false, "Asia", "Asia/Kuwait"},
	{"KY", "🇰🇾", 0, "Cayman Islands", false, "Americas", "America/Cayman"},
	{"LA", "🇱🇦", 0, "Lao People's Democratic Republic", false, "Asia", "Asia/Vientiane"},
	{"LB", "🇱🇧", 0, "Lebanon", false, "Asia", "Asia/Beirut"},
	{"LC", "🇱🇨", 0, "Saint Lucia", false, "Americas", "America/St_Lucia"},
	{"LI", "🇱🇮", 0, "Liechtenstein", false, "Europe", "Europe/Vaduz"},
	{"LK", "🇱🇰", 0, "Sri Lanka", false, "Asia", "Asia/Colombo"},
	{"LR", "🇱🇷", 0, "Liberia", false, "Africa", "Africa/Monrovia"},
	{"LS", "🇱🇸", 0, "Lesotho", false, "Africa", "Africa/Maseru"},
	{"LU", "🇱🇺", 0, "Luxembourg", false, "Europe", "Europe/Luxembourg"},
	{"LY", "🇱🇾", 0, "Libya", false, "Africa", "Africa/Tripoli"},
	{"MA", "🇲🇦", 0, "Morocco", false, "Africa", "Africa/Casablanca"},
	{"MC", "🇲🇨", 0, "Monaco", false, "Europe", "Europe/Monaco"},
	{"ME", "🇲🇪", 0, "Montenegro", false, "Europe", "Europe/Podgorica"},
	{"MF", "🇲🇫", 0, "Saint Martin (French part)", false, "Americas", "America/Marigot"},
	{"MG", "🇲🇬", 0, "Madagascar", false, "Africa", "Indian/Antananarivo"},
	{"MH", "🇲🇭", 0, "Marshall Islands", false, "Oceania", "Pacific/Majuro"},
	{"MK", "🇲🇰", 0, "North Macedonia", false, "Europe", "Europe/Skopje"},
	{"ML", "🇲🇱", 0, "Mali", false, "Africa", "Africa/Bamako"},
	{"MM", "🇲🇲", 0, "Myanmar", false, "Asia", "Asia/Yangon"},
	{"MN", "🇲🇳", 0, "Mongolia", false, "Asia", "Asia/Ulaanbaatar"},
	{"MO", "🇲🇴", 0, "Macao", false, "Asia", "Asia/Macau"},
	{"MP", "🇲🇵", 0, "Northern Mariana Islands", false, "Oceania", "Pacific/Saipan"},
	{"MQ", "🇲🇶", 0, "Martinique", false, "Americas", "America/Martinique"},
	{"MR", "🇲🇷", 0, "Mauritania", false, "Africa", "Africa/Nouakchott"},
	{"MS", "🇲🇸", 0, "Montserrat", false, "Americas", "America/Montserrat"},
	{"MT", "🇲🇹", 0, "Malta", false, "Europe", "Europe/Malta"},
	{"MU", "🇲🇺", 0, "Mauritius", false, "Africa", "Indian/Mauritius"},
	{"MV", "🇲🇻", 0, "Maldives", false, "Asia", "Indian/Maldives"},
	{"MW", "🇲🇼", 0, "Malawi", false, "Africa", "Africa/Blantyre"},
	{"MX", "🇲🇽", 0, "Mexico", false, "Americas", "America/Mexico_City"},
	{"MY", "🇲🇾", 0, "Malaysia", false, "Asia", "Asia/Kuala_Lumpur"},
	{"MZ", "🇲🇿", 0, "Mozambique", false, "Africa", "Africa/Maputo"},
	{"NA", "🇳🇦", 0, "Namibia", false, "Africa", "Africa/Windhoek"},
	{"NC", "🇳🇨", 0, "New Caledonia", false, "Oceania", "Pacific/Noumea"},
	{"NE", "🇳🇪", 0, "Niger", false, "Africa", "Africa/Niamey"},
	{"NF", "🇳🇫", 0, "Norfolk Island", false, "Oceania", "Pacific/Norfolk"},
	{"NG", "🇳🇬", 0, "Nigeria", false, "Africa", "Africa/Lagos"},
	{"NI", "🇳🇮", 0, "Nicaragua", false, "Americas", "America/Managua"},
	{"NP", "🇳🇵", 0, "Nepal", false, "Asia", "Asia/Kathmandu"},
	{"NR", "🇳🇷", 0, "Nauru", false, "Oceania", "Pacific/Nauru"},
	{"NU", "🇳🇺", 0, "Niue", false, "Oceania", "Pacific/Niue"},
	{"NZ", "🇳🇿", 0, "New Zealand", false, "Oceania", "Pacific/Auckland"},
	{"OM", "🇴🇲", 0, "Oman", false, "Asia", "Asia/Muscat"},
	{"PA", "🇵🇦", 0, "Panama", false, "Americas", "America/Panama"},
	{"PE", "🇵🇪", 0, "Peru", false, "Americas", "America/Lima"},
	{"PF", "🇵🇫", 0, "French Polynesia", false, "Oceania", "Pacific/Tahiti"},
	{"PG", "🇵🇬", 0, "Papua New Guinea", false, "Oceania", "Pacific/Port_Moresby"},
	{"PH", "🇵🇭", 0, "Philippines", false, "Asia", "Asia/Manila"},
	{"PK", "🇵🇰", 0, "Pakistan", false, "Asia", "Asia/Karachi"},
	{"PM", "🇵🇲", 0, "Saint Pierre and Miquelon", false, "Americas", "America/Miquelon"},
	{"PN", "🇵🇳", 0, "Pitcairn", false, "Oceania", "Pacific/Pitcairn"},
	{"PR", "🇵🇷", 0, "Puerto Rico", false, "Americas", "America/Puerto_Rico"},
	{"PS", "🇵🇸", 0, "Palestine, State of", false, "Asia", "Asia/Gaza"},
	{"PW", "🇵🇼", 0, "Palau", false, "Oceania", "Pacific/Palau"},
	{"PY", "🇵🇾", 0, "Paraguay", false, "Americas", "America/Asuncion"},
	{"QA", "🇶🇦", 0, "Qatar", false, "Asia", "Asia/Qatar"},
	{"RE", "🇷🇪", 0, "Réunion", false, "Africa", "Indian/Reunion"},
	{"RW", "🇷🇼", 0, "Rwanda", false, "Africa", "Africa/Kigali"},
	{"SB", "🇸🇧", 0, "Solomon Islands", false, "Oceania", "Pacific/Guadalcanal"},
	{"SC", "🇸🇨", 0, "Seychelles", false, "Africa", "Indian/Mahe"},
	{"SD", "🇸🇩", 0, "Sudan", false, "Africa", "Africa/Khartoum"},
	{"SG", "🇸🇬", 0, "Singapore", false, "Asia", "Asia/Singapore"},
	{"SH", "🇸🇭", 0, "Saint Helena, Ascension and Tristan da Cunha", false, "Africa", "Atlantic/St_Helena"},
	{"SI", "🇸🇮", 0, "Slovenia", false, "Europe", "Europe/Ljubljana"},
	{"SJ", "🇸🇯", 0, "Svalbard and Jan Mayen", false, "Europe", "Arctic/Longyearbyen"},
	{"SL", "🇸🇱", 0, "Sierra Leone", false, "Africa", "Africa/Freetown"},
	{"SM", "🇸🇲", 0, "San Marino", false, "Europe", "Europe/San_Marino"},
	{"SN", "🇸🇳", 0, "Senegal", false, "Africa", "Africa/Dakar"},
	{"SO", "🇸🇴", 0, "Somalia", false, "Africa", "Africa/Mogadishu"},
	{"SR", "🇸🇷", 0, "Suriname", false, "Americas", "America/Paramaribo"},
	{"SS", "🇸🇸", 0, "South Sudan", false, "Africa", "Africa/Juba"},
	{"ST", "🇸🇹", 0, "Sao Tome and Principe", false, "Africa", "Africa/Sao_Tome"},
	{"SV", "🇸🇻", 0, "El Salvador", false, "Americas", "America/El_Salvador"},
	{"SX", "🇸🇽", 0, "Sint Maarten (Dutch part)", false, "Americas", "America/Lower_Princes"},
	{"SY", "🇸🇾", 0, "Syrian Arab Republic", false, "Asia", "Asia/Damascus"},
	{"SZ", "🇸🇿", 0, "Eswatini", false, "Africa", "Africa/Mbabane"},
	{"TC", "🇹🇨", 0, "Turks and Caicos Islands", false, "Americas", "America/Grand_Turk"},
	{"TD", "🇹🇩", 0, "Chad", false, "Africa", "Africa/Ndjamena"},
	{"TF", "🇹🇫", 0, "French Southern Territories", false, "Antarctica", "Indian/Kerguelen"},
	{"TG", "🇹🇬", 0, "Togo", false, "Africa", "Africa/Lome"},
	{"TH", "🇹🇭", 0, "Thailand", false, "Asia", "Asia/Bangkok"},
	{"TJ", "🇹🇯", 0, "Tajikistan", false, "Asia", "Asia/Dushanbe"},
	{"TK", "🇹🇰", 0, "Tokelau", false, "Oceania", "Pacific/Fakaofo"},
	{"TL", "🇹🇱", 0, "Timor-Leste", false, "Asia", "Asia/Dili"},
	{"TM", "🇹🇲", 0, "Turkmenistan", false, "Asia", "Asia/Ashgabat"},
	{"TN", "🇹🇳", 0, "Tunisia", false, "Africa", "Africa/Tunis"},
	{"TO", "🇹🇴", 0, "Tonga", false, "Oceania", "Pacific/Tongatapu"},
	{"TT", "🇹🇹", 0, "Trinidad and Tobago", false, "Americas", "America/Port_of_Spain"},
	{"TV", "🇹🇻", 0, "Tuvalu", false, "Oceania", "Pacific/Funafuti"},
	{"TW", "🇹🇼", 0, "Taiwan, Province of China", false, "Asia", "Asia/Taipei"},
	{"TZ", "🇹🇿", 0, "Tanzania, United Republic of", false, "Africa", "Africa/Dar_es_Salaam"},
	{"UG", "🇺🇬", 0, "Uganda", false, "Africa", "Africa/Kampala"},
	{"UM", "🇺🇲", 0, "United States Minor Outlying Islands", false, "Americas", "Pacific/Wake"},
	{"UY", "🇺🇾", 0, "Uruguay", false, "Americas", "America/Montevideo"},
	{"VA", "🇻🇦", 0, "Holy See", false, "Europe", "Europe/Vatican"},
	{"VC", "🇻🇨", 0, "Saint Vincent and the Grenadines", false, "Americas", "America/St_Vincent"},
	{"VE", "🇻🇪", 0, "Venezuela, Bolivarian Republic of", false, "Americas", "America/Caracas"},
	{"VG", "🇻🇬", 0, "Virgin Islands (British)", false, "Americas", "America/Tortola"},
	{"VI", "🇻🇮", 0, "Virgin Islands (U.S.)", false, "Americas", "America/St_Thomas"},
	{"VN", "🇻🇳", 0, "Viet Nam", false, "Asia", "Asia/Ho_Chi_Minh"},
	{"VU", "🇻🇺", 0, "Vanuatu", false, "Oceania", "Pacific/Efate"},
	{"WF", "🇼🇫", 0, "Wallis and Futuna", false, "Oceania", "Pacific/Wallis"},
	{"WS", "🇼🇸", 0, "Samoa", false, "Oceania", "Pacific/Apia"},
	{"YE", "🇾🇪", 0, "Yemen", false, "Asia", "Asia/Aden"},
	{"YT", "🇾🇹", 0, "Mayotte", false, "Africa", "Indian/Mayotte"},
	{"ZA", "🇿🇦", 0, "South Africa", false, "Africa", "Africa/Johannesburg"},
	{"ZM", "🇿🇲", 0, "Zambia", false, "Africa", "Africa/Lusaka"},
	{"ZW", "🇿🇼", 0, "Zimbabwe", false, "Africa", "Africa/Harare"},
}

// GetCountryByCode returns the full Country struct for a given country code
// Returns nil if country code is not found
func GetCountryByCode(countryCode string) *Country {
	code := strings.ToUpper(countryCode)
	for _, country := range Countries {
		if country.Code == code {
			return &country
		}
	}
	return nil
}

// GetFavoriteCountries возвращает список избранных стран
func GetFavoriteCountries() []Country {
	var favorites []Country
	for _, country := range Countries {
		if country.Favorite {
			favorites = append(favorites, country)
		}
	}
	return favorites
}

// GetCountriesByRegion возвращает список стран по региону
func GetCountriesByRegion(region string) []Country {
	var filteredCountries []Country
	for _, country := range Countries {
		if country.Region == region {
			filteredCountries = append(filteredCountries, country)
		}
	}
	return filteredCountries
}

// GetTimezone returns the timezone string for a given country code
// Returns "UTC" if country code is not found
func GetTimezone(countryCode string) string {
	code := strings.ToUpper(countryCode)
	for _, country := range Countries {
		if country.Code == code {
			return country.Timezone
		}
	}
	return "UTC" // Return UTC if not found
}

// GetTimezoneLocation returns *time.Location for a given country code
// Returns UTC if country code is not found or timezone is invalid
func GetTimezoneLocation(countryCode string) *time.Location {
	timezone := GetTimezone(countryCode)
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.UTC
	}
	return loc
}
