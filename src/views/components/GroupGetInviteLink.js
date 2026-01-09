export default {
    name: 'GroupGetInviteLink',
    components: {},
    data() {
        return {
            group_id: '',
            inviteLink: '',
            resetLink: false,
            loading: false,
            copying: false,
        }
    },
    computed: {
        fullGroupID() {
            if (!this.group_id) return '';
            // Ensure suffix
            if (this.group_id.endsWith(window.TYPEGROUP)) {
                return this.group_id;
            }
            return this.group_id + window.TYPEGROUP;
        },
        isValidForm() {
            return this.group_id.trim() !== '';
        },
        displayGroupID() {
            if (!this.group_id) return '';
            // Show the full ID with suffix for clarity
            return this.fullGroupID;
        }
    },
    methods: {
        openModal() {
            this.reset();
            $('#modalGroupGetInviteLink').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        async handleSubmit() {
            if (!this.isValidForm || this.loading) return;
            try {
                await this.getInviteLink();
                if (this.inviteLink) {
                    showSuccessInfo('Link de convite do grupo obtido com sucesso!');
                } else {
                    showErrorInfo('Nenhum link de convite recebido do servidor');
                }
            } catch (err) {
                showErrorInfo(err.message || err);
            }
        },
        async getInviteLink() {
            this.loading = true;
            try {
                const response = await window.http.get(`/group/invite-link`, {
                    params: { 
                        group_id: this.fullGroupID,
                        reset: this.resetLink 
                    }
                });
                if (response.data.results && typeof response.data.results.invite_link === 'string') {
                    this.inviteLink = response.data.results.invite_link;
                } else if (response.data && typeof response.data.invite_link === 'string') {
                    this.inviteLink = response.data.invite_link;
                } else if (typeof response.data.results === 'string') {
                    this.inviteLink = response.data.results;
                } else if (typeof response.data === 'string') {
                    this.inviteLink = response.data;
                } else {
                    this.inviteLink = '';
                }
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message || error.response.data);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        reset() {
            this.group_id = '';
            this.inviteLink = '';
            this.resetLink = false;
            this.loading = false;
            this.copying = false;
        },
        async copyToClipboard() {
            if (this.inviteLink) {
                this.copying = true;
                try {
                    // Try modern clipboard API first
                    if (navigator.clipboard && navigator.clipboard.writeText) {
                        await navigator.clipboard.writeText(this.inviteLink);
                        showSuccessInfo('✅ Link de convite copiado para a área de transferência!');
                    } else {
                        // Fallback for older browsers
                        this.fallbackCopyToClipboard();
                    }
                } catch (err) {
                    console.error('Clipboard API failed:', err);
                    this.fallbackCopyToClipboard();
                }
            } else {
                showErrorInfo('Nenhum link de convite para copiar');
            }
        },
        fallbackCopyToClipboard() {
            try {
                // Create a temporary input element
                const tempInput = document.createElement('input');
                tempInput.style.position = 'absolute';
                tempInput.style.left = '-9999px';
                tempInput.value = this.inviteLink;
                document.body.appendChild(tempInput);
                
                // Select and copy
                tempInput.select();
                tempInput.setSelectionRange(0, 99999); // For mobile devices
                
                const successful = document.execCommand('copy');
                document.body.removeChild(tempInput);
                
                if (successful) {
                    showSuccessInfo('✅ Link de convite copiado para a área de transferência!');
                } else {
                    showErrorInfo('❌ Falha ao copiar. Por favor, selecione e copie manualmente.');
                }
            } catch (err) {
                console.error('Fallback copy failed:', err);
                showErrorInfo('❌ Falha ao copiar. Por favor, selecione e copie manualmente.');
            }
        },
        closeModal() {
            $('#modalGroupGetInviteLink').modal('hide');
        },
        handleGroupIDInput() {
            // Auto-correct the input if it's just numbers
            const input = this.group_id.trim();
            if (input && !input.includes('@') && !input.includes('g.us')) {
                // If it's just numbers, assume it's a group ID
                this.group_id = input + window.TYPEGROUP;
            }
        }
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer;">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Obter Link de Convite do Grupo</div>
            <div class="description">
                Obter link de convite para um grupo
            </div>
        </div>
    </div>

    <!-- Modal -->
    <div class="ui modal" id="modalGroupGetInviteLink">
        <i class="close icon"></i>
        <div class="header">
            <i class="linkify icon"></i>
            Obter Link de Convite do Grupo
        </div>
        <div class="content">
            <form class="ui form">
                <div class="field">
                    <label>ID do Grupo</label>
                    <input type="text" v-model="group_id" 
                           placeholder="Insira o ID do grupo (ex: 120363419080717833)"
                           @blur="handleGroupIDInput"
                           @input="handleGroupIDInput">
                    <div class="ui tiny info message">
                        <i class="info circle icon"></i>
                        <div class="content">
                            <p>Você pode inserir apenas os números (ex: 120363419080717833) - o sufixo @g.us será adicionado automaticamente.</p>
                        </div>
                    </div>
                    <div v-if="group_id && displayGroupID !== group_id" class="ui info message">
                        <i class="info circle icon"></i>
                        <div class="content">
                            <div class="header">ID do Grupo Corrigido Automaticamente</div>
                            <p>Sua entrada: <code>{{ group_id }}</code></p>
                            <p>Será usado: <code>{{ displayGroupID }}</code></p>
                        </div>
                    </div>
                </div>
                
                <div class="field">
                    <div class="ui checkbox">
                        <input type="checkbox" v-model="resetLink" id="resetInviteLink">
                        <label for="resetInviteLink">Redefinir link de convite (revogar link antigo)</label>
                    </div>
                </div>

                <div class="ui divider"></div>

                <div v-if="inviteLink" class="field">
                    <label>Link de Convite</label>
                    <div class="ui action input">
                        <input type="text" :value="inviteLink" readonly 
                               style="font-family: monospace; background-color: #f8f9fa; cursor: text;"
                               @click="$event.target.select()"
                               @focus="$event.target.select()">
                        <button type="button" class="ui blue button" @click="copyToClipboard" :class="{'loading': copying}">
                            <i class="copy icon"></i>
                            <span v-if="copying">Copiando...</span>
                            <span v-else>Copiar</span>
                        </button>
                    </div>
                    <div class="ui info message">
                        <i class="info circle icon"></i>
                        <div class="content">
                            <div class="header">Link de Convite Gerado</div>
                            <p>Compartilhe este link com outras pessoas para convidá-las para o grupo.</p>
                        </div>
                    </div>
                </div>

                <div class="ui divider"></div>

                <button type="button" class="ui approve positive right labeled icon button" 
                        :class="{'loading': loading, 'disabled': !isValidForm || loading}"
                        @click.prevent="handleSubmit">
                    <span v-if="loading">Buscando...</span>
                    <span v-else>Obter Link de Convite</span>
                    <i class="linkify icon"></i>
                </button>
            </form>
        </div>
        <div class="actions">
            <div class="ui approve button" @click="closeModal">Fechar</div>
        </div>
    </div>
    `
} 