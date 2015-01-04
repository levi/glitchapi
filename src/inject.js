(function(app, ads, hls, player, eventController) {

    function makeAppId() {
        var text = "";
        var sample = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefhijklmnopqrstuvwxyz";
        for (var i = 0; i < 26; i++) {
            text += sample.charAt(Math.floor(Math.random() * sample.length));
        }
        return text;
    }

    function appNotify(object, event, opts) {
        opts || (opts = {});
        opts.object = object;
        opts.event = event;
        app.postMessage(opts);
    }

    function cleanAdURL(url) {
        url = url.replace('twitch/channels', 'twitch.m/ios');
        url = url.replace('xml_vast3', 'xml_vast2');
        url = url.replace('platform%3Dhtml5', 'platform%3Diphone');
        url = url.replace('%26pos%3Dpre', '%26pos%3D1');
        return url;
    }

    var AppId = makeAppId();
    appNotify("app", "start", { app_id: AppId });


    window[AppId] = {
        playing: false,
        trackingMinutesWatched: false,
        events: {
            startAd: function() {
                var app = window[AppId];
                jQuery(app.adsController).trigger('start');
            },

            finishAd: function() {
                var app = window[AppId];
                jQuery(app.adsController).trigger('contentResumeRequested');
            },

            playStream: function(eventController) {
                eventController.sendVideoPlay();
                setInterval(function() {
                    appNotify("events", "minute_watched");
                    eventController.sendMinuteWatched(t++);
                }, 6000);
            },

            pauseStream: function() {
                window[AppId].playing = false;
            }
        }
    };

    // Prevent hls prefetching, since it has already been completed
    // within the application
    hls.getPlaylist = function(ch) {
        return {
            then: function(success, err) {
                appNotify("app", "get_playlist");
                success("stream_result");
            }
        }
    };

    // Override player behaviour to noop
    player.prototype.setSrc = function() {};
    player.prototype.play = function() {};
    player.prototype.pause = function() {};
    player.prototype.preloadContent = function() {};

    // Override event controller to notify app of triggers
    var trackVideoPlayingSuper = eventController.prototype.trackVideoPlaying;
    eventController.prototype.trackVideoPlaying = function (e) {
        appNotify("event_controller", "track_video_playing");
        trackVideoPlayingSuper.call(this, e);
    };

    var sendMinuteWatchedSuper = eventController.prototype.sendMinuteWatched;
    eventController.prototype.sendMinuteWatched = function (t) {
        appNotify("event_controller", "minute_watched", {
            minutes_logged: t
        });
        sendMinuteWatchedSuper.call(this, t);
    };

    var sendVideoPlaySuper = eventController.prototype.sendVideoPlay;
    eventController.prototype.sendVideoPlay = function () {
        appNotify("event_controller", "video_play", {
            type: type,
            provider: provider
        });
        sendVideoPlaySuper.call(this);
    };

    var sendVideoAdOpportunitySuper = eventController.prototype.sendVideoAdOpportunity;
    eventController.prototype.sendVideoAdOpportunity = function (type, provider) {
        appNotify("event_controller", "video_ad_opportunity", {
            type: type,
            provider: provider
        });

        var t = 1;
        this.sendVideoPlay();
        setInterval(function() {
            this.sendMinuteWatched(t++);
        }, 60000);
        sendVideoAdOpportunitySuper.call(this, type, provider);
    };

    var sendVideoAdImpressionSuper = eventController.prototype.sendVideoAdImpression;
    eventController.prototype.sendVideoAdImpression = function(type, provider) {
        appNotify("event_controller", "video_ad_impression", {
            type: type,
            provider: provider
        });
        sendVideoAdImpressionSuper.call(this, type, provider);
    };

    // Notify the app of the requested ad URL
    ads.prototype.requestAds = function(url, provider) {
        window[AppId].adsController = this;
        appNotify("ad", "request", { url: cleanAdURL(url) });
    };

})(window.webkit.messageHandlers.GlitchNotification, Twitch.player.AdsController, Twitch.hls, Twitch.player.HTML5Player, Twitch.player.EventController);
