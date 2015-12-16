(function() {
	var interfaces = $('#interfaces');

	var interfaceTemplate = $('#interface_template > div.row');
	var setheaderTemplate = $('#setheader_template > div.row');

	function ReqJSON(method, url, callback, data) {
		var request = {
			type: method,
			url: url,
			success: function(data, textStatus, jqXHR) {
				callback(data)
			},
            error: function(jqXHR, textStatus) {
                if(jqXHR.status==401)
                	alert("Permission denied, please contact your team leader or someone in systems\nServer said:" + jqXHR.responseText)
                if(jqXHR.status==500)
                	alert("Internal server error, please contact systems\nServer said:" + jqXHR.responseText)
            },
			dataType: 'json'
		}
		if (method == 'POST') {
			request.contentType = "application/json; charset=utf-8"
			request.data = JSON.stringify(data);
		}
		jQuery.ajax(request)
	}

	function Interface(ip, data, elm) {
		this.ip = ip;
		this.elm = elm;
		this.elm.attr('id', ip)
			.data('obj', this)
			.find('.ip')
			.text(ip);
		this.elm.find('.description').text(data.description);
		interfaces.append(this.elm);
		this.bssw = this.elm.find('input.switch').bootstrapSwitch().on('switchChange.bootstrapSwitch', function(event, state) {
			this.setEnable(state);
		}.bind(this));
		this.ebtn = this.elm.find('button.extend').on('click', function() {
			this.setEnable(true);
		}.bind(this));

		ReqJSON("GET", "/proxy/" + ip, this.dataRefresh.bind(this));

		return this;
	}

	Interface.prototype.dataRefresh = function(data) {
		this.data = data;

		this.bssw.bootstrapSwitch('disabled', data.TargetURL === null ? true : false, true)
		this.bssw.bootstrapSwitch('state', data.Enabled, true);

		this.elm.find('span.targeturl').text(data.TargetURL === null ? "not forwarded" : data.TargetURL)
		this.elm.find('span.comment').text(data.Comment)
		this.elm.find('span.who').text(data.TargetURL === null || data.Who === "" ? "nobody" : data.Who)
		this.elm.find('span.expire').text(data.TargetURL === null || data.Expire === undefined || data.Expire === "" ? "never" : data.Expire)
		this.elm.find('span.maintainhost').text(data.MaintainHost ? "yes" : "no")
		this.elm.find('button.extend').toggle(data.Enabled)
		var div = this.elm.find('div.setheaders').empty()
		Object.keys(data.SetHeader).forEach(function(name) {
			template = setheaderTemplate.clone(true)
			template.find("div.name").html(name + ':')
			template.find("div.value").html(data.SetHeader[name].join('<br/>'))
			div.append(template)
		}.bind(this))
	}

	Interface.prototype.post = function(target, data) {
		path = "/proxy/" + this.ip;
		if (typeof(target) === "string") {
			path += "/" + target
		}

		ReqJSON("POST", path, this.dataRefresh.bind(this), data)
	};

	Interface.prototype.setEnable = function(enable) {
		this.post('enable', enable)
	};

	function addInterfaces(data) {
		Object.keys(data).forEach(function(k) {
			new Interface(k, data[k], interfaceTemplate.clone(true))
		})
	}

	ReqJSON("GET", "/interfaces", addInterfaces)

	$('#setupModal').on('show.bs.modal', function(event) {
		var btn = $(event.relatedTarget)
		var iface = btn.closest('div.row').data('obj')
		var modal = $(this)

		modal.find('form')[0].reset()

		var targeturl = modal.find('#TargetURL')
		var comment = modal.find('#Comment')
		var maintainhost = modal.find("#MaintainHost")[0]
		var setheader = modal.find('#SetHeader')

		targeturl.val(iface.data.TargetURL)
		comment.val(iface.data.Comment)
		maintainhost.checked = iface.data.MaintainHost
		Object.keys(iface.data.SetHeader).sort().forEach(function (name) {
			setheader.val(setheader.val() + name + ": " + iface.data.SetHeader[name] + "\n")
		})
		setheader.val(setheader.val().trim())

		modal.find('.modal-title').text('Configuring ' + iface.ip)
		modal.find('.btn-primary').off('click').on('click', function() {
			data = {
				TargetURL: targeturl.val(),
				Comment: comment.val(),
				MaintainHost: maintainhost.checked,
				SetHeader: {}
			};

			setheader.val().split(/\r?\n/).forEach(function(header) {
				if (header === "") return;
				nva = header.split(/\s*:\s*/);
				if (nva.length < 2) return;
				data.SetHeader[nva[0].trim()] = [nva[1].trim()];
			})
			iface.post(undefined, data);
			modal.modal('hide');
		})
	})

}())
