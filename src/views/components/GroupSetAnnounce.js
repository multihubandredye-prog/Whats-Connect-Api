export default {
    name: 'GroupSetAnnounce',
    data() {
        return {
            loading: false,
            groupId: '',
            announce: false,
        }
    },
    methods: {
        openModal() {
            $('#modalGroupSetAnnounce').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            return this.groupId.trim() !== '';
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                let response = await this.submitApi()
                showSuccessInfo(response)
                $('#modalGroupSetAnnounce').modal('hide');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let response = await window.http.post(`/group/announce`, {
                    group_id: this.groupId,
                    announce: this.announce
                })
                this.handleReset();
                return response.data.message;
            } catch (error) {
                if (error.response) {
                    throw new Error(error.response.data.message);
                }
                throw new Error(error.message);
            } finally {
                this.loading = false;
            }
        },
        handleReset() {
            this.groupId = '';
            this.announce = false;
        },
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Definir Anúncio do Grupo</div>
            <div class="description">
                Habilitar/desabilitar modo de anúncio para mensagens apenas de administradores
            </div>
        </div>
    </div>
    
    <!--  Modal Group Set Announce  -->
    <div class="ui small modal" id="modalGroupSetAnnounce">
        <i class="close icon"></i>
        <div class="header">
            Definir Modo de Anúncio do Grupo
        </div>
        <div class="content">
            <form class="ui form">
                <div class="field">
                    <label>ID do Grupo</label>
                    <input v-model="groupId" type="text"
                           placeholder="120363024512399999@g.us"
                           aria-label="Group ID">
                </div>
                
                <div class="field">
                    <label>Modo Anúncio</label>
                    <div class="ui toggle checkbox">
                        <input type="checkbox" v-model="announce">
                        <label>{{ announce ? 'Ativar modo anúncio (apenas administradores podem enviar mensagens)' : 'Desativar modo anúncio (todos os membros podem enviar mensagens)' }}</label>
                    </div>
                    <div class="ui info message" style="margin-top: 10px;">
                        <div class="header">O que isso faz?</div>
                        <ul class="list">
                            <li><strong>Modo Anúncio LIGADO:</strong> Apenas administradores do grupo podem enviar mensagens para o grupo</li>
                            <li><strong>Modo Anúncio DESLIGADO:</strong> Todos os membros do grupo podem enviar mensagens</li>
                        </ul>
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui approve positive right labeled icon button" 
                    :class="{'loading': this.loading, 'disabled': !this.isValidForm() || this.loading}"
                    @click.prevent="handleSubmit" type="button">
                {{ announce ? 'Ativar Modo Anúncio' : 'Desativar Modo Anúncio' }}
                <i :class="announce ? 'bullhorn icon' : 'comment icon'"></i>
            </button>
        </div>
    </div>
    `
} 