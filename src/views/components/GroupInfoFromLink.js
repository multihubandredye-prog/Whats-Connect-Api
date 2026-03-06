export default {
    name: 'GroupInfoFromLink',
    data() {
        return {
            loading: false,
            link: '',
            groupInfo: null,
        }
    },
    methods: {
        openModal() {
            $('#modalGroupInfoFromLink').modal({
                onApprove: function () {
                    return false;
                }
            }).modal('show');
        },
        isValidForm() {
            if (!this.link.trim()) {
                return false;
            }

            // should be a valid WhatsApp invitation URL
            try {
                const url = new URL(this.link);
                if (!url.hostname.includes('chat.whatsapp.com') || !url.pathname.includes('/')) {
                    return false;
                }
            } catch (error) {
                return false;
            }

            return true;
        },
        async handleSubmit() {
            if (!this.isValidForm() || this.loading) {
                return;
            }
            try {
                let response = await this.submitApi()
                this.groupInfo = response.results;
                showSuccessInfo('Informações do grupo obtidas com sucesso');
            } catch (err) {
                showErrorInfo(err)
            }
        },
        async submitApi() {
            this.loading = true;
            try {
                let response = await window.http.get(`/group/info-from-link`, {
                    params: {
                        link: this.link,
                    }
                })
                return response.data;
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
            this.link = '';
            this.groupInfo = null;
        },
        formatDate(dateString) {
            if (!dateString) return 'N/D';
            return moment(dateString).format('YYYY-MM-DD HH:mm');
        },
        closeModal() {
            $('#modalGroupInfoFromLink').modal('hide');
            this.handleReset();
        },
    },
    template: `
    <div class="green card" @click="openModal" style="cursor: pointer">
        <div class="content">
            <a class="ui green right ribbon label">Grupo</a>
            <div class="header">Pré-visualização do Grupo</div>
            <div class="description">
                Obter informações do grupo a partir do link de convite
            </div>
        </div>
    </div>
    
    <!--  Modal Group Info From Link  -->
    <div class="ui small modal" id="modalGroupInfoFromLink">
        <i class="close icon"></i>
        <div class="header">
            Pré-visualização das Informações do Grupo
        </div>
        <div class="content">
            <form class="ui form">
                <div class="field">
                    <label>Link de Convite</label>
                    <input v-model="link" type="text"
                           placeholder="Link de convite..."
                           aria-label="Link de Convite">
                </div>
                
                <div v-if="groupInfo" class="ui segment">
                    <h4 class="ui header">Detalhes do Grupo</h4>
                    <div class="ui relaxed divided list">
                        <div class="item">
                            <div class="content">
                                <div class="header">Nome do Grupo</div>
                                <div class="description">{{ groupInfo.name || 'N/D' }}</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">ID do Grupo</div>
                                <div class="description">{{ groupInfo.group_id || 'N/D' }}</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">Tópico</div>
                                <div class="description">{{ groupInfo.topic || 'Nenhum tópico definido' }}</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">Descrição</div>
                                <div class="description">{{ groupInfo.description || 'Nenhuma descrição' }}</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">Criado Em</div>
                                <div class="description">{{ formatDate(groupInfo.created_at) }}</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">Participantes</div>
                                <div class="description">{{ groupInfo.participant_count || 0 }} membros</div>
                            </div>
                        </div>
                        <div class="item">
                            <div class="content">
                                <div class="header">Configurações do Grupo</div>
                                <div class="description">
                                    <div class="ui mini labels">
                                        <div class="ui label" :class="groupInfo.is_locked ? 'red' : 'green'">
                                            <i class="lock icon"></i>
                                            {{ groupInfo.is_locked ? 'Bloqueado' : 'Desbloqueado' }}
                                        </div>
                                        <div class="ui label" :class="groupInfo.is_announce ? 'orange' : 'blue'">
                                            <i class="bullhorn icon"></i>
                                            {{ groupInfo.is_announce ? 'Modo Anúncio' : 'Modo Regular' }}
                                        </div>
                                        <div class="ui label" :class="groupInfo.is_ephemeral ? 'purple' : 'grey'">
                                            <i class="clock icon"></i>
                                            {{ groupInfo.is_ephemeral ? 'Mensagens Temporárias' : 'Mensagens Regulares' }}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </form>
        </div>
        <div class="actions">
            <button class="ui grey button" @click="closeModal">
                Fechar
            </button>
            <button class="ui approve positive right labeled icon button" 
                    :class="{'loading': this.loading, 'disabled': !this.isValidForm() || this.loading}"
                    @click.prevent="handleSubmit" type="button">
                Obter Informações
                <i class="info icon"></i>
            </button>
        </div>
    </div>
    `
}