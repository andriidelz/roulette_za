-- Локализации на английском языке
INSERT INTO localizations (key, language, value) VALUES
    ('balanceaccok', 'en', 'Your balance is %.2f USDT. The amount is sufficient for withdrawal. You can make a withdrawal in the Withdraw menu item.'),
    ('balanceacclow', 'en', 'Your balance is %.2f USDT. The amount is insufficient for withdrawal. Play more to enter the top players of the week and distribute the prize fund!'),
    
    ('withdrawok', 'en', 'Your balance is %.2f USDT. You can make a withdrawal by continuing.'),
    ('withdrawlow', 'en', 'Your balance is %.2f USDT. The amount is insufficient for withdrawal. Play more to enter the top players of the week and distribute the prize fund!'),
    
    ('zero_limit', 'en', 'You cannot bet on Zero yet, which can bring 10 points to the rating. You need to make %d more bets today. Until then, if Zero comes up, it counts as a loss for you'),
    
    ('nomorebids', 'en', 'Bets accepted! No more bets!'),
    ('nextbid15', 'en', 'Round #%s

There are 15 seconds left until the next bet determination.

Make your choice'),
    ('nextbid5', 'en', 'Round #%s

There are 5 seconds left until the next bet determination.

Make your choice'),
    
    ('blackresult', 'en', 'Black on the roulette!'),
    ('redresult', 'en', 'Red on the roulette!'),
    ('zeroresult', 'en', 'Zero on the roulette!'),
    
    ('bet_error', 'en', 'Error when betting. Please try again.'),
    ('error', 'en', 'An error occurred. Please try again later.'),
    ('bet_already_made', 'en', 'You have already made a bet in this round. Please wait for the result.'),
    
    ('btn_play', 'en', '🎮 Play'),
    ('btn_stats', 'en', '📊 Statistics'),
    ('btn_rating', 'en', '🏆 Rating'),
    ('btn_balance', 'en', '💰 Balance'),
    ('btn_faq', 'en', '❓ FAQ'),
    ('btn_bet_red', 'en', '🔴 Red'),
    ('btn_bet_black', 'en', '⚫️ Black'),
    ('btn_bet_zero', 'en', '🟢 Zero'),
    ('btn_bet_zero_locked', 'en', '🔒 Zero'),
    ('btn_back', 'en', '◀️ Back'),
    
    ('startmessage1', 'en', 'Welcome to Sprut Red&Black bot

This is the only bot where you can win real money every week for virtual guesses.'),
    ('countrymes', 'en', 'To continue, please select your country of residence. This is necessary for the future localization of player ratings and increasing your chances of winning'),
    ('btn_rules', 'en', 'Rules'),
    ('btn_awards', 'en', 'Rewards'),
    ('btn_payments', 'en', 'Payments'),
    ('btn_fairplay', 'en', 'Fair Play'),
    ('btn_statistics', 'en', 'Statistics'),
    ('btn_account', 'en', 'Account'),
    ('rules', 'en', 'Game Rules:

1. You place a bet on red, black, or zero.
2. Every 30 seconds, a random number from 0 to 36 is determined.
3. Numbers from 1 to 36 correspond to red or black.
4. 0 corresponds to zero.
5. For each correct prediction, you earn points.
6. Your points are counted in the weekly ranking.'),
    ('awards', 'en', 'Rewards:

1. A player ranking is formed every week.
2. The top 100 players share the prize pool.
3. The prize pool is distributed in proportion to the points earned.
4. Payments are processed automatically every Monday.
5. Minimum withdrawal amount: 10 USDT.'),
    ('payments', 'en', 'Payments:

1. All prizes are paid in USDT (TRC-20).
2. To receive payment, you need to specify your wallet address.
3. Minimum withdrawal amount: 10 USDT.
4. Withdrawal requests are processed within 24 hours.'),
    ('fairplay', 'en', 'Fair Play:

Roulette results are guaranteed to be fair and verifiable. Before each round, a hash of the result is published. After the round, you can verify that the result was not changed.

To verify:
1. Take the number and salt provided by the bot.
2. Form a string in the format: [number]:[salt]
3. Calculate the SHA-256 hash of this string.
4. Compare with the hash from the beginning of the round.'),
    ('main_menu', 'en', 'Tap a button to continue'),
    ('country_saved', 'en', 'Your country has been successfully saved! ✅'),
    ('settings_message', 'en', 'Settings Menu

Here you can change your profile settings:'),
    ('btn_settings_language', 'en', '🌐 Language'),
    ('btn_settings_country', 'en', '🌍 Country'),
    ('btn_settings_nickname', 'en', '👤 Nickname'),
    ('btn_settings_name', 'en', '👤 First Name'),
    ('btn_back_to_main', 'en', '◀️ Back to Main Menu'),
    ('settings_language', 'en', 'Select your language:'),
    ('settings_nickname', 'en', 'Enter your nickname:'),
    ('settings_name', 'en', 'Enter your first name:'),
    ('language_saved', 'en', 'Language successfully updated! ✅'),
    ('name_saved', 'en', 'First name successfully updated! ✅'),
    ('nickname_saved', 'en', 'Nickname successfully updated! ✅'),
    ('btn_settings_wallet', 'en', '💰 USDT Wallet Address'),
    ('settings_wallet', 'en', 'Enter your USDT wallet address (TRC20):'),
    ('wallet_saved', 'en', 'Wallet address successfully updated! ✅'),
    ('invalid_wallet_format', 'en', 'Invalid wallet address format. Please enter a valid TRC20 wallet address starting with T.'),
    ('unknown_command', 'en', 'Unknown command. Please use the menu to navigate.'),
    ('statisticsstart', 'en', 'Your personal game statistics are available in this section. Select the period for which you want to view statistics'),
    ('daystat', 'en', 'Today'),
    ('weekstat', 'en', 'Weekly'),
    ('monthstat', 'en', 'Monthly'),
    ('allstat', 'en', 'All-time'),
    ('exitstat', 'en', 'Main menu'),
    ('daystatm', 'en', E'Your statistics for today\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
    ('weekstatm', 'en', E'Your statistics for the current week\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
    ('monthstatm', 'en', E'Your statistics for the current month\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
    ('allstatm', 'en', E'Your statistics for all time\nMade %d bets (%d black, %d red, %d ZERO)\nof which\nWon %d bets (%d black, %d red, %d ZERO)\nLost %d bets (%d black, %d red, %d ZERO)\n\nYou earned %d rating points'),
    ('statistics next', 'en', 'Select another time period to get statistics or return to the main menu'),
    ('playstart1', 'en', E'Game essence\nYour task is to guess the color of the field that will appear on the virtual roulette.\n\nFor each correct guess you get 1 credit point...'),
    ('rulesstart', 'en', 'Detailed Rules'),
    ('availablebets', 'en', 'Available bets'),
    ('stop', 'en', 'Stop game'),
    ('betsbalancelow', 'en', 'You have run out of bets for today. Come back tomorrow!'),
    ('betsbalanceok', 'en', 'You have %d more bets available today.'),
    ('round_info_countdown', 'en', E'Round #%s\nHash: %s\n\n%d seconds left until the next bet determination.\n\nMake your choice'),
    ('waiting_for_round', 'en', 'Waiting for a new round to start. Please wait...'),
    ('systemcheck', 'en', E'Check round in the system'),
    ('viewrating', 'en', E'View rating'),
    ('topupbalance', 'en', E'Refill attempts balance'),
    ('stopgame', 'en', E'Stop game'),
    ('nextbidlow', 'en', E'There are 5 seconds left until the next bet determination.\n\nYou can no longer play. You have no more bets'),
    ('winmessage', 'en', E'You won!\nYou have been credited with %d rating points'),
    ('losemessage', 'en', E'You lost!'),
    ('bidrating', 'en', E'You currently have %d rating points\nYou are in position %d in the weekly ranking and are eligible for %.2f $ from the prize fund of %.2f $'),
    ('startmessage2', 'en', 'To participate in the weekly prize fund competition absolutely free, subscribe to our reserve channel to stay connected. The reserve channel will only be active when there are issues with access to the main bot.'),
    ('go_to_channel', 'en', 'Go to channel'),
    ('reservsubs', 'en', 'I have subscribed'),
    ('reservok', 'en', 'Your subscription has been confirmed! Now you can start playing and earning'),
    ('reservno', 'en', 'You probably haven''t subscribed to the channel and can''t start playing. Subscribe to the reserve channel and request verification again.'),
    ('faqstart', 'en', 'In this section, you can read the rules for participating in the bot, privacy policy, and find answers to the most common questions from participants'),
    ('faqrules', 'en', 'Rules'),
    ('faqawards', 'en', 'Prize Distribution'),
    ('faqpayments', 'en', 'Prize Payments'),
    ('faqfairplay', 'en', 'Fair Play Principles'),
    ('privacypolicy', 'en', 'Privacy Policy'),
    ('contact', 'en', 'Contact Admin'),
    ('faqexit', 'en', 'Main Menu'),
    ('faqrulesm', 'en', '-'),
    ('faqawardsm', 'en', '-'),
    ('faqpaymentsm', 'en', '-'),
    ('faqfairplaym', 'en', 'Fair Play Principles:

Our roulette uses cryptographic verification to ensure result fairness:

1. Before each round, a hash of the result is published
2. The result cannot be changed after the round begins
3. After the round, the number and salt are revealed
4. Players can verify the hash matches the result

Verification steps:
- Take the number and salt provided after the round
- Create a string: [number]:[salt]
- Calculate the SHA-256 hash
- Compare with the hash provided at the beginning of the round

This ensures 100% fair and unpredictable results.'),
    ('privacypolicym', 'en', 'Privacy Policy:

1. Data we collect:
   - Telegram ID and username
   - Name and language settings from Telegram
   - Country (selected by you)
   - Cryptocurrency wallet address (for payments)
   - Game statistics and betting history

2. How we use your data:
   - To provide gaming services
   - To calculate ratings and distribute prizes
   - To process payments
   - To prevent fraud and abuse

3. Data protection:
   - We use industry-standard security measures
   - We do not share your personal data with third parties
   - We may use anonymized data for analytics

4. Your rights:
   - Access to your personal data
   - Correction of inaccurate data
   - Deletion of your account and data

For questions regarding privacy, contact the administrator.'),
    ('contactm', 'en', 'Here you can leave your suggestion or question, and we will contact you. Please write to our administrator: @roulette_admin'),
    ('faqnext', 'en', 'Choose the next section for information or return to the main menu'),
    ('ratingstart', 'en', 'In this section you can view the current rating
Select a rating type to view'),
    ('weekrat', 'en', 'Weekly top'),
    ('personalrat', 'en', 'Your position'),
    ('exitrat', 'en', 'Main menu'),
    ('weekratm', 'en', 'Current anonymized ranking of players who will share the reward for the current week
Note: we anonymize player names and only show the number of rating points to demonstrate how many rating points are currently needed to participate in the distribution of the reward.'),
    ('personalratm', 'en', 'Your place in the overall ranking'),
    ('ratingnext', 'en', 'Select another rating type to display or return to the main menu'),
    ('weekly_rating_empty', 'en', 'Current anonymized ranking of players who will share the reward for the current week
Note: we anonymize player names and only show the number of rating points to demonstrate how many rating points are currently needed to participate in the distribution of the reward.

The rating is currently empty. Be the first to earn points!'),
    ('weekly_rating_top', 'en', 'Current anonymized ranking of players who will share the reward for the current week
Note: we anonymize player names and only show the number of rating points to demonstrate how many rating points are currently needed to participate in the distribution of the reward.

Top %d players:

%s'),
    ('weekly_rating_all', 'en', 'Current anonymized ranking of players who will share the reward for the current week
Note: we anonymize player names and only show the number of rating points to demonstrate how many rating points are currently needed to participate in the distribution of the reward.

Current rating:

%s'),
    ('personal_rating_empty', 'en', 'Your place in the overall ranking

Your position: %d

The rating is currently empty. Be the first to earn points!'),
    ('personal_rating_prize_zone', 'en', 'Your place in the overall ranking

Your position: %d

%s

Congratulations! You are in the prize zone and will take part in the distribution of the weekly reward.'),
    ('personal_rating_need_points', 'en', 'Your place in the overall ranking

Your position: %d

%s

You need %d more points to participate in the distribution of the weekly reward.'),
    ('player_points', 'en', '%d %s - %d points'),
    ('player_points_efficiency', 'en', '%d %s - %d points (%.1f%%)'),
    ('username_points', 'en', '*%s* - %d points'),
    ('username_points_efficiency', 'en', '*%s* - %d points (%.1f%%)'),
    ('accstart', 'en', 'In this section you can manage your personal account'),
    ('balance', 'en', 'Balance'),
    ('withdraw', 'en', 'Withdraw'),
    ('bonus', 'en', 'Bonuses'),
    ('buybets', 'en', 'Convert to attempts'),
    ('exitacc', 'en', 'Main menu'),
    ('balaccokwith', 'en', 'Request withdrawal'),
    ('balancenext', 'en', 'Select the next action with your personal account or return to the main menu'),
    ('withdrawproc', 'en', 'Request withdrawal'),
    ('withdrawlownext', 'en', 'Select the next action with your personal account or return to the main menu'),
    ('withdrawusdtcheck', 'en', 'The USDT withdrawal address specified in your account is %s. Please check and confirm the address or change it. Sending USDT to an invalid address cannot be undone.'),
    ('usdtok', 'en', 'Accept address'),
    ('usdtchange', 'en', 'Change address'),
    ('no_wallet_address', 'en', 'To withdraw funds, you need to specify a wallet address in your profile settings.'),
    ('go_to_settings', 'en', 'Go to settings'),
    ('insufficient_balance', 'en', 'Insufficient amount for withdrawal. Minimum amount: 10 USDT'),
    ('withdrawal_error', 'en', 'Error creating withdrawal request. Please try again later.'),
    ('withdrawal_success', 'en', 'Withdrawal request for %.2f USDT successfully created. Funds will be sent to wallet %s within 24 hours.');

INSERT INTO localizations (key, language, value) VALUES
    ('agemes', 'en', 'Please confirm that you are over 18 years old'),
    ('yes18', 'en', 'Yes, I am over 18 years old'),
    ('no18', 'en', 'No, I am under 18 years old'),
    ('stopage', 'en', 'Sorry, this service is only available to users over 18 years old.'),
    ('balance_updated_title', 'en', 'Reward credited to your balance'),
    ('balance_updated_message', 'en', 'You have received a reward of {amount} to your internal balance!'),
    ('top_rating_title', 'en', 'Weekly Top Rating'),
    ('top_rating_message', 'en', 'Congratulations! You are now in the weekly top rating at position {position}!'),
    ('name_mes', 'en', 'Your name that will be displayed in the general rating is {profile_name} and will be publicly visible to all players. Do you want to change it?'),
    ('name_changeyes', 'en', 'Yes, I want to change'),
    ('name_changeno', 'en', 'No, I don''t want to change'),
    ('name_changeno_msg', 'en', 'Your game name has been fixed'),
    ('name_changeok', 'en', 'Enter the desired name for public display in the rating. Only Latin alphabet, numbers and underscore are allowed. The name must be 3-20 characters long.'),
    ('name_changesave', 'en', 'New game name for rating has been saved.'),
    ('invalid_nickname', 'en', 'Invalid nickname. The nickname should contain only Latin letters, numbers, and underscores, and be 3-20 characters long. Please try again.'),
    ('stopcountry', 'en', 'Service is not available for residents of Russia or Belarus.');

INSERT INTO localizations (key, language, value) VALUES
    ('error_retrieving_balance', 'en', 'Error retrieving balance. Please try again.'),
    ('error_retrieving_data', 'en', 'Error retrieving data. Please try again.'),
    ('error_while_registering', 'en', 'Error while registering. Please try again.'),
    ('player_nickname_template', 'en', 'Nickname%d');

INSERT INTO localizations (key, language, value) VALUES
    ('rating_error', 'en', 'Error retrieving rating data. Please try again later.');
