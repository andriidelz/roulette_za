package data

import (
	"strings"
	"time"
)

// Country представляет информацию о стране
type Country struct {
	Code     string // ISO 3166-1 alpha-2
	Emoji    string
	Name     string
	Favorite bool
	Region   string
	Timezone string
}

// Countries содержит список стран
var Countries = []Country{

	// Favorite

	{"AE", "🇦🇪", "United Arab Emirates", true, "Asia", "Asia/Dubai"},
	{"AU", "🇦🇺", "Australia", true, "Oceania", "Australia/Sydney"},
	{"BR", "🇧🇷", "Brazil", true, "Americas", "America/Sao_Paulo"},
	{"CY", "🇨🇾", "Cyprus", true, "Europe", "Asia/Nicosia"},
	{"CZ", "🇨🇿", "Czechia", true, "Europe", "Europe/Prague"},
	{"DE", "🇩🇪", "Germany", true, "Europe", "Europe/Berlin"},
	{"DK", "🇩🇰", "Denmark", true, "Europe", "Europe/Copenhagen"},
	{"EE", "🇪🇪", "Estonia", true, "Europe", "Europe/Tallinn"},
	{"EG", "🇪🇬", "Egypt", true, "Africa", "Africa/Cairo"},
	{"ES", "🇪🇸", "Spain", true, "Europe", "Europe/Madrid"},
	{"FI", "🇫🇮", "Finland", true, "Europe", "Europe/Helsinki"},
	{"FR", "🇫🇷", "France", true, "Europe", "Europe/Paris"},
	{"GB", "🇬🇧", "United Kingdom of Great Britain and Northern Ireland", true, "Europe", "Europe/London"},
	{"GE", "🇬🇪", "Georgia", true, "Asia", "Asia/Tbilisi"},
	{"GR", "🇬🇷", "Greece", true, "Europe", "Europe/Athens"},
	{"HR", "🇭🇷", "Croatia", true, "Europe", "Europe/Zagreb"},
	{"HU", "🇭🇺", "Hungary", true, "Europe", "Europe/Budapest"},
	{"IL", "🇮🇱", "Israel", true, "Asia", "Asia/Jerusalem"},
	{"IN", "🇮🇳", "India", true, "Asia", "Asia/Kolkata"},
	{"IT", "🇮🇹", "Italy", true, "Europe", "Europe/Rome"},
	{"KZ", "🇰🇿", "Kazakhstan", true, "Asia", "Asia/Almaty"},
	{"LT", "🇱🇹", "Lithuania", true, "Europe", "Europe/Vilnius"},
	{"LV", "🇱🇻", "Latvia", true, "Europe", "Europe/Riga"},
	{"MD", "🇲🇩", "Moldova, Republic of Moldova", true, "Europe", "Europe/Chisinau"},
	{"NL", "🇳🇱", "Netherlands, Kingdom of the Netherlands", true, "Europe", "Europe/Amsterdam"},
	{"NO", "🇳🇴", "Norway", true, "Europe", "Europe/Oslo"},
	{"PL", "🇵🇱", "Poland", true, "Europe", "Europe/Warsaw"},
	{"PT", "🇵🇹", "Portugal", true, "Europe", "Europe/Lisbon"},
	{"RO", "🇷🇴", "Romania", true, "Europe", "Europe/Bucharest"},
	{"RS", "🇷🇸", "Serbia", true, "Europe", "Europe/Belgrade"},
	{"SA", "🇸🇦", "Saudi Arabia", true, "Asia", "Asia/Riyadh"},
	{"SE", "🇸🇪", "Sweden", true, "Europe", "Europe/Stockholm"},
	{"SK", "🇸🇰", "Slovakia", true, "Europe", "Europe/Bratislava"},
	{"TR", "🇹🇷", "Turkey, Republic of Türkiye", true, "Asia", "Europe/Istanbul"},
	{"UA", "🇺🇦", "Ukraine", true, "Europe", "Europe/Kiev"},
	{"US", "🇺🇸", "United States of America", true, "Americas", "America/New_York"},
	{"UZ", "🇺🇿", "Uzbekistan", true, "Asia", "Asia/Tashkent"},

	// Non favorite

	{"AD", "🇦🇩", "Andorra", false, "Europe", "Europe/Andorra"},
	{"AF", "🇦🇫", "Afghanistan", false, "Asia", "Asia/Kabul"},
	{"AG", "🇦🇬", "Antigua and Barbuda", false, "Americas", "America/Antigua"},
	{"AI", "🇦🇮", "Anguilla", false, "Americas", "America/Anguilla"},
	{"AL", "🇦🇱", "Albania", false, "Europe", "Europe/Tirane"},
	{"AM", "🇦🇲", "Armenia", false, "Asia", "Asia/Yerevan"},
	{"AO", "🇦🇴", "Angola", false, "Africa", "Africa/Luanda"},
	{"AQ", "🇦🇶", "Antarctica", false, "Antarctica", "Antarctica/McMurdo"},
	{"AR", "🇦🇷", "Argentina", false, "Americas", "America/Argentina/Buenos_Aires"},
	{"AS", "🇦🇸", "American Samoa", false, "Oceania", "Pacific/Pago_Pago"},
	{"AT", "🇦🇹", "Austria", false, "Europe", "Europe/Vienna"},
	{"AW", "🇦🇼", "Aruba", false, "Americas", "America/Aruba"},
	{"AX", "🇦🇽", "Åland Islands", false, "Europe", "Europe/Mariehamn"},
	{"AZ", "🇦🇿", "Azerbaijan", false, "Asia", "Asia/Baku"},
	{"BA", "🇧🇦", "Bosnia and Herzegovina", false, "Europe", "Europe/Sarajevo"},
	{"BB", "🇧🇧", "Barbados", false, "Americas", "America/Barbados"},
	{"BD", "🇧🇩", "Bangladesh", false, "Asia", "Asia/Dhaka"},
	{"BE", "🇧🇪", "Belgium", false, "Europe", "Europe/Brussels"},
	{"BF", "🇧🇫", "Burkina Faso", false, "Africa", "Africa/Ouagadougou"},
	{"BG", "🇧🇬", "Bulgaria", false, "Europe", "Europe/Sofia"},
	{"BH", "🇧🇭", "Bahrain", false, "Asia", "Asia/Bahrain"},
	{"BI", "🇧🇮", "Burundi", false, "Africa", "Africa/Bujumbura"},
	{"BJ", "🇧🇯", "Benin", false, "Africa", "Africa/Porto-Novo"},
	{"BL", "🇧🇱", "Saint Barthélemy", false, "Americas", "America/St_Barthelemy"},
	{"BM", "🇧🇲", "Bermuda", false, "Americas", "Atlantic/Bermuda"},
	{"BN", "🇧🇳", "Brunei Darussalam", false, "Asia", "Asia/Brunei"},
	{"BO", "🇧🇴", "Bolivia, Plurinational State of", false, "Americas", "America/La_Paz"},
	{"BQ", "🇧🇶", "Bonaire, Sint Eustatius and Saba", false, "Americas", "America/Kralendijk"},
	{"BS", "🇧🇸", "Bahamas", false, "Americas", "America/Nassau"},
	{"BT", "🇧🇹", "Bhutan", false, "Asia", "Asia/Thimphu"},
	{"BV", "🇧🇻", "Bouvet Island", false, "Antarctica", "Antarctica/Syowa"},
	{"BW", "🇧🇼", "Botswana", false, "Africa", "Africa/Gaborone"},
	{"BY", "❌", "Belarus", false, "Europe", "Europe/Minsk"},
	{"BZ", "🇧🇿", "Belize", false, "Americas", "America/Belize"},
	{"CA", "🇨🇦", "Canada", false, "Americas", "America/Toronto"},
	{"CC", "🇨🇨", "Cocos (Keeling) Islands", false, "Asia", "Indian/Cocos"},
	{"CD", "🇨🇩", "Congo, Democratic Republic of the", false, "Africa", "Africa/Kinshasa"},
	{"CF", "🇨🇫", "Central African Republic", false, "Africa", "Africa/Bangui"},
	{"CG", "🇨🇬", "Congo", false, "Africa", "Africa/Brazzaville"},
	{"CH", "🇨🇭", "Switzerland", false, "Europe", "Europe/Zurich"},
	{"CI", "🇨🇮", "Côte d'Ivoire", false, "Africa", "Africa/Abidjan"},
	{"CK", "🇨🇰", "Cook Islands", false, "Oceania", "Pacific/Rarotonga"},
	{"CL", "🇨🇱", "Chile", false, "Americas", "America/Santiago"},
	{"CM", "🇨🇲", "Cameroon", false, "Africa", "Africa/Douala"},
	{"CN", "🇨🇳", "China", false, "Asia", "Asia/Shanghai"},
	{"CO", "🇨🇴", "Colombia", false, "Americas", "America/Bogota"},
	{"CR", "🇨🇷", "Costa Rica", false, "Americas", "America/Costa_Rica"},
	{"CU", "🇨🇺", "Cuba", false, "Americas", "America/Havana"},
	{"CV", "🇨🇻", "Cabo Verde", false, "Africa", "Atlantic/Cape_Verde"},
	{"CW", "🇨🇼", "Curaçao", false, "Americas", "America/Curacao"},
	{"CX", "🇨🇽", "Christmas Island", false, "Asia", "Indian/Christmas"},
	{"DJ", "🇩🇯", "Djibouti", false, "Africa", "Africa/Djibouti"},
	{"DM", "🇩🇲", "Dominica", false, "Americas", "America/Dominica"},
	{"DO", "🇩🇴", "Dominican Republic", false, "Americas", "America/Santo_Domingo"},
	{"DZ", "🇩🇿", "Algeria", false, "Africa", "Africa/Algiers"},
	{"EC", "🇪🇨", "Ecuador", false, "Americas", "America/Guayaquil"},
	{"EH", "🇪🇭", "Western Sahara", false, "Africa", "Africa/El_Aaiun"},
	{"ER", "🇪🇷", "Eritrea", false, "Africa", "Africa/Asmara"},
	{"ET", "🇪🇹", "Ethiopia", false, "Africa", "Africa/Addis_Ababa"},
	{"FJ", "🇫🇯", "Fiji", false, "Oceania", "Pacific/Fiji"},
	{"FK", "🇫🇰", "Falkland Islands (Malvinas)", false, "Americas", "Atlantic/Stanley"},
	{"FM", "🇫🇲", "Micronesia, Federated States of", false, "Oceania", "Pacific/Chuuk"},
	{"FO", "🇫🇴", "Faroe Islands", false, "Europe", "Atlantic/Faroe"},
	{"GA", "🇬🇦", "Gabon", false, "Africa", "Africa/Libreville"},
	{"GD", "🇬🇩", "Grenada", false, "Americas", "America/Grenada"},
	{"GF", "🇬🇫", "French Guiana", false, "Americas", "America/Cayenne"},
	{"GG", "🇬🇬", "Guernsey", false, "Europe", "Europe/Guernsey"},
	{"GH", "🇬🇭", "Ghana", false, "Africa", "Africa/Accra"},
	{"GI", "🇬🇮", "Gibraltar", false, "Europe", "Europe/Gibraltar"},
	{"GL", "🇬🇱", "Greenland", false, "Americas", "America/Nuuk"},
	{"GM", "🇬🇲", "Gambia", false, "Africa", "Africa/Banjul"},
	{"GN", "🇬🇳", "Guinea", false, "Africa", "Africa/Conakry"},
	{"GP", "🇬🇵", "Guadeloupe", false, "Americas", "America/Guadeloupe"},
	{"GQ", "🇬🇶", "Equatorial Guinea", false, "Africa", "Africa/Malabo"},
	{"GS", "🇬🇸", "South Georgia and the South Sandwich Islands", false, "Antarctica", "Atlantic/South_Georgia"},
	{"GT", "🇬🇹", "Guatemala", false, "Americas", "America/Guatemala"},
	{"GU", "🇬🇺", "Guam", false, "Oceania", "Pacific/Guam"},
	{"GW", "🇬🇼", "Guinea-Bissau", false, "Africa", "Africa/Bissau"},
	{"GY", "🇬🇾", "Guyana", false, "Americas", "America/Guyana"},
	{"HK", "🇭🇰", "Hong Kong", false, "Asia", "Asia/Hong_Kong"},
	{"HM", "🇭🇲", "Heard Island and McDonald Islands", false, "Antarctica", "Indian/Kerguelen"},
	{"HN", "🇭🇳", "Honduras", false, "Americas", "America/Tegucigalpa"},
	{"HT", "🇭🇹", "Haiti", false, "Americas", "America/Port-au-Prince"},
	{"ID", "🇮🇩", "Indonesia", false, "Asia", "Asia/Jakarta"},
	{"IE", "🇮🇪", "Ireland", false, "Europe", "Europe/Dublin"},
	{"IM", "🇮🇲", "Isle of Man", false, "Europe", "Europe/Isle_of_Man"},
	{"IO", "🇮🇴", "British Indian Ocean Territory", false, "Asia", "Indian/Chagos"},
	{"IQ", "🇮🇶", "Iraq", false, "Asia", "Asia/Baghdad"},
	{"IR", "🇮🇷", "Iran, Islamic Republic of", false, "Asia", "Asia/Tehran"},
	{"IS", "🇮🇸", "Iceland", false, "Europe", "Atlantic/Reykjavik"},
	{"JE", "🇯🇪", "Jersey", false, "Europe", "Europe/Jersey"},
	{"JM", "🇯🇲", "Jamaica", false, "Americas", "America/Jamaica"},
	{"JO", "🇯🇴", "Jordan", false, "Asia", "Asia/Amman"},
	{"JP", "🇯🇵", "Japan", false, "Asia", "Asia/Tokyo"},
	{"KE", "🇰🇪", "Kenya", false, "Africa", "Africa/Nairobi"},
	{"KG", "🇰🇬", "Kyrgyzstan", false, "Asia", "Asia/Bishkek"},
	{"KH", "🇰🇭", "Cambodia", false, "Asia", "Asia/Phnom_Penh"},
	{"KI", "🇰🇮", "Kiribati", false, "Oceania", "Pacific/Tarawa"},
	{"KM", "🇰🇲", "Comoros", false, "Africa", "Indian/Comoro"},
	{"KN", "🇰🇳", "Saint Kitts and Nevis", false, "Americas", "America/St_Kitts"},
	{"KP", "🇰🇵", "Korea, Democratic People's Republic of", false, "Asia", "Asia/Pyongyang"},
	{"KR", "🇰🇷", "Korea, Republic of", false, "Asia", "Asia/Seoul"},
	{"KW", "🇰🇼", "Kuwait", false, "Asia", "Asia/Kuwait"},
	{"KY", "🇰🇾", "Cayman Islands", false, "Americas", "America/Cayman"},
	{"LA", "🇱🇦", "Lao People's Democratic Republic", false, "Asia", "Asia/Vientiane"},
	{"LB", "🇱🇧", "Lebanon", false, "Asia", "Asia/Beirut"},
	{"LC", "🇱🇨", "Saint Lucia", false, "Americas", "America/St_Lucia"},
	{"LI", "🇱🇮", "Liechtenstein", false, "Europe", "Europe/Vaduz"},
	{"LK", "🇱🇰", "Sri Lanka", false, "Asia", "Asia/Colombo"},
	{"LR", "🇱🇷", "Liberia", false, "Africa", "Africa/Monrovia"},
	{"LS", "🇱🇸", "Lesotho", false, "Africa", "Africa/Maseru"},
	{"LU", "🇱🇺", "Luxembourg", false, "Europe", "Europe/Luxembourg"},
	{"LY", "🇱🇾", "Libya", false, "Africa", "Africa/Tripoli"},
	{"MA", "🇲🇦", "Morocco", false, "Africa", "Africa/Casablanca"},
	{"MC", "🇲🇨", "Monaco", false, "Europe", "Europe/Monaco"},
	{"ME", "🇲🇪", "Montenegro", false, "Europe", "Europe/Podgorica"},
	{"MF", "🇲🇫", "Saint Martin (French part)", false, "Americas", "America/Marigot"},
	{"MG", "🇲🇬", "Madagascar", false, "Africa", "Indian/Antananarivo"},
	{"MH", "🇲🇭", "Marshall Islands", false, "Oceania", "Pacific/Majuro"},
	{"MK", "🇲🇰", "North Macedonia", false, "Europe", "Europe/Skopje"},
	{"ML", "🇲🇱", "Mali", false, "Africa", "Africa/Bamako"},
	{"MM", "🇲🇲", "Myanmar", false, "Asia", "Asia/Yangon"},
	{"MN", "🇲🇳", "Mongolia", false, "Asia", "Asia/Ulaanbaatar"},
	{"MO", "🇲🇴", "Macao", false, "Asia", "Asia/Macau"},
	{"MP", "🇲🇵", "Northern Mariana Islands", false, "Oceania", "Pacific/Saipan"},
	{"MQ", "🇲🇶", "Martinique", false, "Americas", "America/Martinique"},
	{"MR", "🇲🇷", "Mauritania", false, "Africa", "Africa/Nouakchott"},
	{"MS", "🇲🇸", "Montserrat", false, "Americas", "America/Montserrat"},
	{"MT", "🇲🇹", "Malta", false, "Europe", "Europe/Malta"},
	{"MU", "🇲🇺", "Mauritius", false, "Africa", "Indian/Mauritius"},
	{"MV", "🇲🇻", "Maldives", false, "Asia", "Indian/Maldives"},
	{"MW", "🇲🇼", "Malawi", false, "Africa", "Africa/Blantyre"},
	{"MX", "🇲🇽", "Mexico", false, "Americas", "America/Mexico_City"},
	{"MY", "🇲🇾", "Malaysia", false, "Asia", "Asia/Kuala_Lumpur"},
	{"MZ", "🇲🇿", "Mozambique", false, "Africa", "Africa/Maputo"},
	{"NA", "🇳🇦", "Namibia", false, "Africa", "Africa/Windhoek"},
	{"NC", "🇳🇨", "New Caledonia", false, "Oceania", "Pacific/Noumea"},
	{"NE", "🇳🇪", "Niger", false, "Africa", "Africa/Niamey"},
	{"NF", "🇳🇫", "Norfolk Island", false, "Oceania", "Pacific/Norfolk"},
	{"NG", "🇳🇬", "Nigeria", false, "Africa", "Africa/Lagos"},
	{"NI", "🇳🇮", "Nicaragua", false, "Americas", "America/Managua"},
	{"NP", "🇳🇵", "Nepal", false, "Asia", "Asia/Kathmandu"},
	{"NR", "🇳🇷", "Nauru", false, "Oceania", "Pacific/Nauru"},
	{"NU", "🇳🇺", "Niue", false, "Oceania", "Pacific/Niue"},
	{"NZ", "🇳🇿", "New Zealand", false, "Oceania", "Pacific/Auckland"},
	{"OM", "🇴🇲", "Oman", false, "Asia", "Asia/Muscat"},
	{"PA", "🇵🇦", "Panama", false, "Americas", "America/Panama"},
	{"PE", "🇵🇪", "Peru", false, "Americas", "America/Lima"},
	{"PF", "🇵🇫", "French Polynesia", false, "Oceania", "Pacific/Tahiti"},
	{"PG", "🇵🇬", "Papua New Guinea", false, "Oceania", "Pacific/Port_Moresby"},
	{"PH", "🇵🇭", "Philippines", false, "Asia", "Asia/Manila"},
	{"PK", "🇵🇰", "Pakistan", false, "Asia", "Asia/Karachi"},
	{"PM", "🇵🇲", "Saint Pierre and Miquelon", false, "Americas", "America/Miquelon"},
	{"PN", "🇵🇳", "Pitcairn", false, "Oceania", "Pacific/Pitcairn"},
	{"PR", "🇵🇷", "Puerto Rico", false, "Americas", "America/Puerto_Rico"},
	{"PS", "🇵🇸", "Palestine, State of", false, "Asia", "Asia/Gaza"},
	{"PW", "🇵🇼", "Palau", false, "Oceania", "Pacific/Palau"},
	{"PY", "🇵🇾", "Paraguay", false, "Americas", "America/Asuncion"},
	{"QA", "🇶🇦", "Qatar", false, "Asia", "Asia/Qatar"},
	{"RE", "🇷🇪", "Réunion", false, "Africa", "Indian/Reunion"},
	{"RU", "❌", "Russian Federation", false, "Europe", "Europe/Moscow"},
	{"RW", "🇷🇼", "Rwanda", false, "Africa", "Africa/Kigali"},
	{"SB", "🇸🇧", "Solomon Islands", false, "Oceania", "Pacific/Guadalcanal"},
	{"SC", "🇸🇨", "Seychelles", false, "Africa", "Indian/Mahe"},
	{"SD", "🇸🇩", "Sudan", false, "Africa", "Africa/Khartoum"},
	{"SG", "🇸🇬", "Singapore", false, "Asia", "Asia/Singapore"},
	{"SH", "🇸🇭", "Saint Helena, Ascension and Tristan da Cunha", false, "Africa", "Atlantic/St_Helena"},
	{"SI", "🇸🇮", "Slovenia", false, "Europe", "Europe/Ljubljana"},
	{"SJ", "🇸🇯", "Svalbard and Jan Mayen", false, "Europe", "Arctic/Longyearbyen"},
	{"SL", "🇸🇱", "Sierra Leone", false, "Africa", "Africa/Freetown"},
	{"SM", "🇸🇲", "San Marino", false, "Europe", "Europe/San_Marino"},
	{"SN", "🇸🇳", "Senegal", false, "Africa", "Africa/Dakar"},
	{"SO", "🇸🇴", "Somalia", false, "Africa", "Africa/Mogadishu"},
	{"SR", "🇸🇷", "Suriname", false, "Americas", "America/Paramaribo"},
	{"SS", "🇸🇸", "South Sudan", false, "Africa", "Africa/Juba"},
	{"ST", "🇸🇹", "Sao Tome and Principe", false, "Africa", "Africa/Sao_Tome"},
	{"SV", "🇸🇻", "El Salvador", false, "Americas", "America/El_Salvador"},
	{"SX", "🇸🇽", "Sint Maarten (Dutch part)", false, "Americas", "America/Lower_Princes"},
	{"SY", "🇸🇾", "Syrian Arab Republic", false, "Asia", "Asia/Damascus"},
	{"SZ", "🇸🇿", "Eswatini", false, "Africa", "Africa/Mbabane"},
	{"TC", "🇹🇨", "Turks and Caicos Islands", false, "Americas", "America/Grand_Turk"},
	{"TD", "🇹🇩", "Chad", false, "Africa", "Africa/Ndjamena"},
	{"TF", "🇹🇫", "French Southern Territories", false, "Antarctica", "Indian/Kerguelen"},
	{"TG", "🇹🇬", "Togo", false, "Africa", "Africa/Lome"},
	{"TH", "🇹🇭", "Thailand", false, "Asia", "Asia/Bangkok"},
	{"TJ", "🇹🇯", "Tajikistan", false, "Asia", "Asia/Dushanbe"},
	{"TK", "🇹🇰", "Tokelau", false, "Oceania", "Pacific/Fakaofo"},
	{"TL", "🇹🇱", "Timor-Leste", false, "Asia", "Asia/Dili"},
	{"TM", "🇹🇲", "Turkmenistan", false, "Asia", "Asia/Ashgabat"},
	{"TN", "🇹🇳", "Tunisia", false, "Africa", "Africa/Tunis"},
	{"TO", "🇹🇴", "Tonga", false, "Oceania", "Pacific/Tongatapu"},
	{"TT", "🇹🇹", "Trinidad and Tobago", false, "Americas", "America/Port_of_Spain"},
	{"TV", "🇹🇻", "Tuvalu", false, "Oceania", "Pacific/Funafuti"},
	{"TW", "🇹🇼", "Taiwan, Province of China", false, "Asia", "Asia/Taipei"},
	{"TZ", "🇹🇿", "Tanzania, United Republic of", false, "Africa", "Africa/Dar_es_Salaam"},
	{"UG", "🇺🇬", "Uganda", false, "Africa", "Africa/Kampala"},
	{"UM", "🇺🇲", "United States Minor Outlying Islands", false, "Americas", "Pacific/Wake"},
	{"UY", "🇺🇾", "Uruguay", false, "Americas", "America/Montevideo"},
	{"VA", "🇻🇦", "Holy See", false, "Europe", "Europe/Vatican"},
	{"VC", "🇻🇨", "Saint Vincent and the Grenadines", false, "Americas", "America/St_Vincent"},
	{"VE", "🇻🇪", "Venezuela, Bolivarian Republic of", false, "Americas", "America/Caracas"},
	{"VG", "🇻🇬", "Virgin Islands (British)", false, "Americas", "America/Tortola"},
	{"VI", "🇻🇮", "Virgin Islands (U.S.)", false, "Americas", "America/St_Thomas"},
	{"VN", "🇻🇳", "Viet Nam", false, "Asia", "Asia/Ho_Chi_Minh"},
	{"VU", "🇻🇺", "Vanuatu", false, "Oceania", "Pacific/Efate"},
	{"WF", "🇼🇫", "Wallis and Futuna", false, "Oceania", "Pacific/Wallis"},
	{"WS", "🇼🇸", "Samoa", false, "Oceania", "Pacific/Apia"},
	{"YE", "🇾🇪", "Yemen", false, "Asia", "Asia/Aden"},
	{"YT", "🇾🇹", "Mayotte", false, "Africa", "Indian/Mayotte"},
	{"ZA", "🇿🇦", "South Africa", false, "Africa", "Africa/Johannesburg"},
	{"ZM", "🇿🇲", "Zambia", false, "Africa", "Africa/Lusaka"},
	{"ZW", "🇿🇼", "Zimbabwe", false, "Africa", "Africa/Harare"},
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
