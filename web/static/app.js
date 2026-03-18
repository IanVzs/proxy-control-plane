let providers = [];
let services = [];
let todayUsage = {};
let currentServiceName = '';

document.addEventListener('DOMContentLoaded', function() {
    loadProviders();
    loadServices();
    setupNavigation();
    setupModals();
    setupProxyTest();
});

function setupNavigation() {
    const navBtns = document.querySelectorAll('.nav-btn');
    const tabContents = document.querySelectorAll('.tab-content');

    navBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            navBtns.forEach(b => b.classList.remove('active'));
            tabContents.forEach(t => t.classList.remove('active'));

            this.classList.add('active');
            const tab = this.dataset.tab;
            document.getElementById(tab).classList.add('active');
        });
    });
}

function setupModals() {
    document.querySelectorAll('.close').forEach(btn => {
        btn.addEventListener('click', function() {
            const modalId = this.dataset.modal;
            document.getElementById(modalId).classList.remove('active');
        });
    });

    document.querySelectorAll('[data-dismiss]').forEach(btn => {
        btn.addEventListener('click', function() {
            const modalId = this.dataset.dismiss;
            document.getElementById(modalId).classList.remove('active');
        });
    });

    document.querySelectorAll('.modal').forEach(modal => {
        modal.addEventListener('click', function(e) {
            if (e.target === this) {
                this.classList.remove('active');
            }
        });
    });

    document.getElementById('addProviderBtn').addEventListener('click', function() {
        document.getElementById('providerModalTitle').textContent = '添加代理提供商';
        document.getElementById('providerForm').reset();
        document.getElementById('providerId').value = '';
        document.getElementById('providerEnabled').checked = true;
        document.getElementById('providerModal').classList.add('active');
    });

    document.getElementById('providerForm').addEventListener('submit', function(e) {
        e.preventDefault();
        saveProvider();
    });

    document.getElementById('addServiceBtn').addEventListener('click', function() {
        document.getElementById('serviceForm').reset();
        document.getElementById('serviceModal').classList.add('active');
    });

    document.getElementById('serviceForm').addEventListener('submit', function(e) {
        e.preventDefault();
        createService();
    });

    document.getElementById('configServiceEnabled').addEventListener('change', function() {
        updateServiceEnabled(currentServiceName, this.checked);
    });

    document.getElementById('addProviderToServiceBtn').addEventListener('click', function() {
        populateProviderSelect();
        document.getElementById('addProviderToServiceModal').classList.add('active');
    });

    document.getElementById('addProviderToServiceForm').addEventListener('submit', function(e) {
        e.preventDefault();
        addProviderToService();
    });
}

function setupProxyTest() {
    document.getElementById('testProxyBtn').addEventListener('click', testProxy);
}

async function loadProviders() {
    try {
        const response = await fetch('/api/providers');
        const data = await response.json();
        providers = data.providers || [];
        renderProvidersTable();
    } catch (error) {
        console.error('加载提供商失败:', error);
    }
}

async function loadServices() {
    try {
        const response = await fetch('/api/services');
        const data = await response.json();
        services = data.services || [];
        todayUsage = data.today_usage || {};
        renderServicesTable();
    } catch (error) {
        console.error('加载服务失败:', error);
    }
}

function renderProvidersTable() {
    const tbody = document.getElementById('providersBody');
    
    if (providers.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="empty-state">
                    <p>暂无代理提供商</p>
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = providers.map(provider => `
        <tr>
            <td>${escapeHtml(provider.name)}</td>
            <td>${escapeHtml(provider.type)}</td>
            <td>${escapeHtml(provider.host)}:${provider.port}</td>
            <td>${provider.countries && provider.countries.length > 0 
                ? provider.countries.join(', ') 
                : '<span style="color:#999">全部</span>'}</td>
            <td>
                <span class="status-badge ${provider.enabled ? 'status-enabled' : 'status-disabled'}">
                    ${provider.enabled ? '已启用' : '已禁用'}
                </span>
            </td>
            <td>
                <div class="action-buttons">
                    <button class="btn btn-sm btn-primary" onclick="editProvider('${escapeHtml(provider.id)}')">编辑</button>
                    <button class="btn btn-sm btn-danger" onclick="deleteProvider('${escapeHtml(provider.id)}')">删除</button>
                </div>
            </td>
        </tr>
    `).join('');
}

function renderServicesTable() {
    const tbody = document.getElementById('servicesBody');
    
    if (services.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" class="empty-state">
                    <p>暂无服务配置</p>
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = services.map(service => {
        const providerNames = (service.providers || []).map(pw => {
            const provider = providers.find(p => p.id === pw.provider_id);
            return provider ? provider.name : pw.provider_id;
        });

        const todayCount = todayUsage[service.name] || 0;

        return `
            <tr>
                <td><strong>${escapeHtml(service.name)}</strong></td>
                <td class="providers-cell">
                    <div class="providers-cell-list">
                        ${providerNames.length > 0 
                            ? providerNames.map(n => `<span class="provider-tag">${escapeHtml(n)}</span>`).join('')
                            : '<span style="color:#999">无</span>'}
                    </div>
                </td>
                <td class="today-usage-cell">${todayCount}</td>
                <td>${service.request_count || 0}</td>
                <td>
                    <span class="status-badge ${service.enabled ? 'status-enabled' : 'status-disabled'}">
                        ${service.enabled ? '启用' : '禁用'}
                    </span>
                </td>
                <td>
                    <div class="action-buttons">
                        <button class="btn btn-sm btn-primary" onclick="openServiceConfig('${escapeHtml(service.name)}')">配置</button>
                        <button class="btn btn-sm btn-danger" onclick="deleteService('${escapeHtml(service.name)}')">删除</button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

function formatDate(timestamp) {
    if (!timestamp) return 'N/A';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('zh-CN');
}

function escapeHtml(text) {
    if (!text) return '';
    return String(text)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

async function saveProvider() {
    const id = document.getElementById('providerId').value;
    const name = document.getElementById('providerName').value;
    const type = document.getElementById('providerType').value;
    const host = document.getElementById('providerHost').value;
    const port = parseInt(document.getElementById('providerPort').value);
    const username = document.getElementById('providerUsername').value;
    const password = document.getElementById('providerPassword').value;
    const countriesInput = document.getElementById('providerCountries').value;
    const enabled = document.getElementById('providerEnabled').checked;

    const countries = countriesInput.split(',').map(c => c.trim()).filter(c => c);

    const provider = {
        id,
        name,
        type,
        host,
        port,
        username,
        password,
        countries,
        enabled
    };

    try {
        const url = id 
            ? `/api/providers/${encodeURIComponent(id)}`
            : '/api/providers';
        const method = id ? 'PUT' : 'POST';

        const response = await fetch(url, {
            method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(provider)
        });

        if (!response.ok) {
            throw new Error('保存失败');
        }

        document.getElementById('providerModal').classList.remove('active');
        await loadProviders();
        
        alert(id ? '更新成功' : '添加成功');
    } catch (error) {
        alert('保存失败: ' + error.message);
    }
}

async function editProvider(id) {
    const provider = providers.find(p => p.id === id);
    if (!provider) {
        alert('提供商不存在');
        return;
    }

    document.getElementById('providerModalTitle').textContent = '编辑代理提供商';
    document.getElementById('providerId').value = provider.id;
    document.getElementById('providerName').value = provider.name;
    document.getElementById('providerType').value = provider.type;
    document.getElementById('providerHost').value = provider.host;
    document.getElementById('providerPort').value = provider.port;
    document.getElementById('providerUsername').value = provider.username;
    document.getElementById('providerPassword').value = provider.password;
    document.getElementById('providerCountries').value = (provider.countries || []).join(', ');
    document.getElementById('providerEnabled').checked = provider.enabled;

    document.getElementById('providerModal').classList.add('active');
}

async function deleteProvider(id) {
    if (!confirm('确定要删除这个代理提供商吗？')) {
        return;
    }

    try {
        const response = await fetch(`/api/providers/${encodeURIComponent(id)}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('删除失败');
        }

        await loadProviders();
        alert('删除成功');
    } catch (error) {
        alert('删除失败: ' + error.message);
    }
}

async function createService() {
    const name = document.getElementById('serviceName').value;

    try {
        const response = await fetch('/api/services', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ name })
        });

        if (!response.ok) {
            const data = await response.json();
            throw new Error(data.error || '创建失败');
        }

        document.getElementById('serviceModal').classList.remove('active');
        await loadServices();
        alert('创建成功');
    } catch (error) {
        alert('创建失败: ' + error.message);
    }
}

async function deleteService(name) {
    if (!confirm('确定要删除这个服务吗？')) {
        return;
    }

    try {
        const response = await fetch(`/api/services/${encodeURIComponent(name)}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('删除失败');
        }

        await loadServices();
        alert('删除成功');
    } catch (error) {
        alert('删除失败: ' + error.message);
    }
}

async function openServiceConfig(name) {
    currentServiceName = name;
    
    try {
        const response = await fetch(`/api/services/${encodeURIComponent(name)}`);
        const data = await response.json();
        const service = data.service;
        const usage = data.today_usage || {};

        document.getElementById('configServiceName').textContent = service.name;
        document.getElementById('configServiceEnabled').checked = service.enabled;

        renderServiceProvidersList(service.providers || []);
        renderTodayUsageList(usage);

        document.getElementById('serviceConfigModal').classList.add('active');
    } catch (error) {
        alert('加载服务配置失败: ' + error.message);
    }
}

function renderServiceProvidersList(serviceProviders) {
    const container = document.getElementById('serviceProvidersList');

    if (serviceProviders.length === 0) {
        container.innerHTML = '<p style="color:#999;text-align:center;padding:20px;">暂无关联的代理提供商</p>';
        return;
    }

    container.innerHTML = serviceProviders.map(sp => {
        const provider = providers.find(p => p.id === sp.provider_id);
        const providerName = provider ? provider.name : sp.provider_id;

        return `
            <div class="provider-item">
                <div class="provider-item-info">
                    <div class="provider-item-name">${escapeHtml(providerName)}</div>
                    <div class="provider-item-weight">
                        <span>权重:</span>
                        <input type="number" value="${sp.weight}" min="1" 
                            onchange="updateProviderWeight('${escapeHtml(sp.provider_id)}', this.value)">
                    </div>
                </div>
                <div class="provider-item-actions">
                    <button class="btn btn-sm btn-danger" onclick="removeProviderFromService('${escapeHtml(sp.provider_id)}')">移除</button>
                </div>
            </div>
        `;
    }).join('');
}

function renderTodayUsageList(usage) {
    const container = document.getElementById('todayUsageList');
    const entries = Object.entries(usage);

    if (entries.length === 0) {
        container.innerHTML = '<p style="color:#999;text-align:center;padding:20px;">今日暂无使用记录</p>';
        return;
    }

    container.innerHTML = entries.map(([providerId, count]) => {
        const provider = providers.find(p => p.id === providerId);
        const providerName = provider ? provider.name : providerId;

        return `
            <div class="usage-item">
                <span class="usage-item-provider">${escapeHtml(providerName)}</span>
                <span class="usage-item-count">${count} 次</span>
            </div>
        `;
    }).join('');
}

function populateProviderSelect() {
    const select = document.getElementById('selectProviderForService');
    const service = services.find(s => s.name === currentServiceName);
    const existingIds = (service.providers || []).map(p => p.provider_id);

    select.innerHTML = '<option value="">请选择...</option>';
    
    providers.forEach(p => {
        if (!existingIds.includes(p.id)) {
            select.innerHTML += `<option value="${escapeHtml(p.id)}">${escapeHtml(p.name)}</option>`;
        }
    });
}

async function addProviderToService() {
    const providerId = document.getElementById('selectProviderForService').value;
    const weight = parseInt(document.getElementById('providerWeight').value) || 1;

    if (!providerId) {
        alert('请选择代理提供商');
        return;
    }

    try {
        const response = await fetch(`/api/services/${encodeURIComponent(currentServiceName)}/providers`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ provider_id: providerId, weight })
        });

        if (!response.ok) {
            throw new Error('添加失败');
        }

        const data = await response.json();
        
        document.getElementById('addProviderToServiceModal').classList.remove('active');
        
        renderServiceProvidersList(data.providers || []);
        await loadServices();
    } catch (error) {
        alert('添加失败: ' + error.message);
    }
}

async function removeProviderFromService(providerId) {
    if (!confirm('确定要移除这个代理提供商吗？')) {
        return;
    }

    try {
        const response = await fetch(`/api/services/${encodeURIComponent(currentServiceName)}/providers/${encodeURIComponent(providerId)}`, {
            method: 'DELETE'
        });

        if (!response.ok) {
            throw new Error('移除失败');
        }

        const data = await response.json();
        renderServiceProvidersList(data.providers || []);
        await loadServices();
    } catch (error) {
        alert('移除失败: ' + error.message);
    }
}

async function updateProviderWeight(providerId, weight) {
    try {
        const response = await fetch(`/api/services/${encodeURIComponent(currentServiceName)}/providers/${encodeURIComponent(providerId)}/weight`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ weight: parseInt(weight) })
        });

        if (!response.ok) {
            throw new Error('更新失败');
        }

        await loadServices();
    } catch (error) {
        alert('更新权重失败: ' + error.message);
    }
}

async function updateServiceEnabled(name, enabled) {
    try {
        const response = await fetch(`/api/services/${encodeURIComponent(name)}/enabled`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ enabled })
        });

        if (!response.ok) {
            throw new Error('更新失败');
        }

        await loadServices();
    } catch (error) {
        alert('更新状态失败: ' + error.message);
        document.getElementById('configServiceEnabled').checked = !enabled;
    }
}

async function testProxy() {
    const url = document.getElementById('proxyUrl').value;
    const serviceName = document.getElementById('proxyServiceName').value;
    const country = document.getElementById('proxyCountry').value;
    const region = document.getElementById('proxyRegion').value;
    const session = document.getElementById('proxySession').value;
    const duration = document.getElementById('proxyDuration').value;
    const providerId = document.getElementById('proxyProviderId').value;
    const method = document.getElementById('proxyMethod').value;

    if (!url) {
        alert('请输入目标URL');
        return;
    }

    const resultBox = document.getElementById('proxyResult');
    resultBox.innerHTML = '<span class="loading"></span> 正在测试...';
    resultBox.className = 'result-box';

    try {
        const headers = {
            'X-Target-Url': url
        };

        if (serviceName) headers['X-Service-Name'] = serviceName;
        if (country) headers['X-Proxy-Country'] = country;
        if (region) headers['X-Proxy-Region'] = region;
        if (session) headers['X-Proxy-Session'] = session;
        if (duration) headers['X-Proxy-Duration'] = duration;
        if (providerId) headers['X-Proxy-Provider'] = providerId;

        const response = await fetch('/proxy', {
            method,
            headers
        });

        const responseHeaders = {};
        response.headers.forEach((value, key) => {
            responseHeaders[key] = value;
        });

        const statusText = response.ok ? '成功' : '失败';
        const statusColor = response.ok ? 'success' : 'error';

        resultBox.className = `result-box ${statusColor}`;
        resultBox.innerHTML = `
            <strong>状态: ${response.status} ${response.statusText}</strong><br><br>
            <strong>响应头:</strong><br>
            ${Object.entries(responseHeaders).map(([k, v]) => `${k}: ${v}`).join('<br>')}<br><br>
            <strong>响应内容:</strong><br>
            <pre>${escapeHtml(await response.text()).slice(0, 5000)}</pre>
        `;
    } catch (error) {
        resultBox.className = 'result-box error';
        resultBox.innerHTML = `
            <strong>错误:</strong> ${escapeHtml(error.message)}
        `;
    }
}
